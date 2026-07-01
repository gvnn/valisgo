package pypi

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type PyPIProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
}

//go:embed templates/*.html
var templateFS embed.FS

var (
	indexTemplate   = template.Must(template.ParseFS(templateFS, "templates/index.html"))
	packageTemplate = template.Must(template.ParseFS(templateFS, "templates/package.html"))
)

func NewPyPIProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage) *PyPIProtocol {
	return &PyPIProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
	}
}

func (p *PyPIProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/simple/", p.handleSimpleIndex)
	r.Get("/simple/{package}/", p.handleSimplePackage)
	r.Get("/packages/{filename}", p.handleDownload)
	r.Post("/", p.handleUpload)

	return r
}

func (p *PyPIProtocol) handleSimpleIndex(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())

	pkgs, err := p.packageStore.ListByRepository(repo.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	reg := domain.RegistryFromContext(req.Context())

	if acceptsJSON(req) {
		p.serveSimpleIndexJSON(w, pkgs)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		RepoName     string
		RegistryName string
		Packages     []*domain.Package
	}{
		RepoName:     repo.Name,
		RegistryName: reg.Name,
		Packages:     pkgs,
	}

	if err := indexTemplate.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) handleSimplePackage(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	pkgName := chi.URLParam(req, "package")
	normalized := NormalizeName(pkgName)

	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalized, repo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "package not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	files, err := p.packageFileStore.ListByPackage(pkg.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	reg := domain.RegistryFromContext(req.Context())

	if acceptsJSON(req) {
		p.serveSimplePackageJSON(w, reg, repo, pkg, files)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		RepoName     string
		RegistryName string
		PackageName  string
		Files        []*domain.PackageFile
	}{
		RepoName:     repo.Name,
		RegistryName: reg.Name,
		PackageName:  pkg.Name,
		Files:        files,
	}

	if err := packageTemplate.ExecuteTemplate(w, "package.html", data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) serveSimpleIndexJSON(w http.ResponseWriter, pkgs []*domain.Package) {
	w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
	resp := SimpleIndexResponse{
		Meta: SimpleMeta{APIVersion: "1.1"},
	}
	for _, pkg := range pkgs {
		resp.Projects = append(resp.Projects, SimpleProject{Name: pkg.NormalizedName})
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) serveSimplePackageJSON(w http.ResponseWriter, reg *domain.Registry, repo *domain.Repository, pkg *domain.Package, files []*domain.PackageFile) {
	w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
	resp := SimplePackageResponse{
		Meta:  SimpleMeta{APIVersion: "1.1"},
		Name:  pkg.Name,
		Files: []SimpleFile{},
	}

	versionsSet := make(map[string]struct{})
	for _, file := range files {
		if _, exists := versionsSet[file.Version]; !exists {
			versionsSet[file.Version] = struct{}{}
			resp.Versions = append(resp.Versions, file.Version)
		}

		resp.Files = append(resp.Files, SimpleFile{
			Filename:   file.Filename,
			URL:        fmt.Sprintf("/registries/%s/repositories/%s/packages/%s", reg.Name, repo.Name, file.Filename),
			Hashes:     SimpleFileHashes{SHA256: file.Hash},
			Size:       file.Size,
			UploadTime: file.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
		})
	}

	if resp.Versions == nil {
		resp.Versions = []string{}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) handleDownload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	filename := chi.URLParam(req, "filename")

	pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	reader, err := p.storage.Get(req.Context(), pkgFile.BlobKey)
	if err != nil {
		http.Error(w, "failed to read file from storage", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	io.Copy(w, reader)
}

type uploadMetadata struct {
	Name           string
	NormalizedName string
	Version        string
	Filename       string
	Size           int64
	File           io.ReadCloser
}

func parseUploadForm(req *http.Request) (*uploadMetadata, error) {
	err := req.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		return nil, fmt.Errorf("invalid form: %w", err)
	}

	if req.FormValue(":action") != "file_upload" {
		return nil, errors.New("unsupported action")
	}

	name := req.FormValue("name")
	version := req.FormValue("version")
	if name == "" || version == "" {
		return nil, errors.New("missing name or version")
	}

	file, header, err := req.FormFile("content")
	if err != nil {
		return nil, fmt.Errorf("missing file content: %w", err)
	}

	return &uploadMetadata{
		Name:           name,
		NormalizedName: NormalizeName(name),
		Version:        version,
		Filename:       header.Filename,
		Size:           header.Size,
		File:           file,
	}, nil
}

func (p *PyPIProtocol) getOrCreatePackage(ctx context.Context, repoID uint, name, normalizedName string) (*domain.Package, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		pkg = &domain.Package{
			Name:           name,
			NormalizedName: normalizedName,
			RepositoryID:   repoID,
		}
		if err := p.packageStore.Create(pkg); err != nil {
			return nil, fmt.Errorf("failed to create package: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	return pkg, nil
}

func (p *PyPIProtocol) storeFileAndMetadata(ctx context.Context, repoID uint, pkg *domain.Package, meta *uploadMetadata) error {
	hasher := sha256.New()
	tee := io.TeeReader(meta.File, hasher)

	blobKey := fmt.Sprintf("%d/%s", repoID, meta.Filename)
	if err := p.storage.Put(ctx, blobKey, tee); err != nil {
		return fmt.Errorf("failed to store file")
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))

	pkgFile := &domain.PackageFile{
		PackageID: pkg.ID,
		Version:   meta.Version,
		Filename:  meta.Filename,
		Hash:      hashString,
		Size:      meta.Size,
		BlobKey:   blobKey,
	}

	if err := p.packageFileStore.Create(pkgFile); err != nil {
		// Best effort rollback
		_ = p.storage.Delete(ctx, blobKey)
		return fmt.Errorf("failed to save file metadata")
	}

	return nil
}

func (p *PyPIProtocol) handleUpload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())

	meta, err := parseUploadForm(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer meta.File.Close()

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, meta.Name, meta.NormalizedName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = p.packageFileStore.GetByFilenameAndPackage(meta.Filename, pkg.ID)
	if err == nil {
		http.Error(w, "file already exists", http.StatusConflict)
		return
	}

	if err := p.storeFileAndMetadata(req.Context(), repo.ID, pkg, meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
