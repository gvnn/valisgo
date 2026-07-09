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
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/registry"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type PyPIProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
	cacheService     *proxy.CacheService
	downloadSF       singleflight.Group
}

//go:embed templates/*.html
var templateFS embed.FS

var (
	indexTemplate   = template.Must(template.ParseFS(templateFS, "templates/index.html"))
	packageTemplate = template.Must(template.ParseFS(templateFS, "templates/package.html"))
)

func NewPyPIProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage, cacheService *proxy.CacheService) *PyPIProtocol {
	return &PyPIProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
		cacheService:     cacheService,
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
	reg := domain.RegistryFromContext(req.Context())

	slog.Info("Handling PyPI simple index request", "registry", reg.Name, "repository", repo.Name, "type", repo.Type)

	var allPkgs []*domain.Package
	var err error

	if repo.Type == domain.RepositoryTypeVirtual {
		allPkgs, err = p.buildVirtualIndex(repo)
	} else {
		allPkgs, err = p.packageStore.ListByRepository(repo.ID)
	}

	if err != nil {
		slog.Error("Failed to list packages", "error", err, "repository", repo.Name)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if acceptsJSON(req) {
		p.serveSimpleIndexJSON(w, allPkgs)
		return
	}

	p.serveSimpleIndexHTML(w, reg, repo, allPkgs)
}

func (p *PyPIProtocol) buildVirtualIndex(repo *domain.Repository) ([]*domain.Package, error) {
	return p.packageStore.ListDistinctByVirtualRepository(repo.ID)
}

func (p *PyPIProtocol) serveSimpleIndexHTML(w http.ResponseWriter, reg *domain.Registry, repo *domain.Repository, pkgs []*domain.Package) {
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
		slog.Error("Failed to execute index template", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) fetchLocalPackageFiles(req *http.Request, reg *domain.Registry, repo *domain.Repository, normalized string) ([]templateFile, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalized, repo.ID)
	if err != nil {
		return nil, err
	}
	files, err := p.packageFileStore.ListByPackage(pkg.ID)
	if err != nil {
		return nil, err
	}
	var tFiles []templateFile
	for _, f := range files {
		tFiles = append(tFiles, templateFile{
			Filename:   f.Filename,
			Hash:       f.Hash,
			URL:        fmt.Sprintf("/registries/%s/repositories/%s/packages/%s", reg.Name, repo.Name, f.Filename),
			Size:       f.Size,
			UploadTime: f.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
			Version:    f.Version,
		})
	}
	return tFiles, nil
}

func (p *PyPIProtocol) handleSimplePackage(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	pkgName := chi.URLParam(req, "package")
	normalized := NormalizeName(pkgName)
	reg := domain.RegistryFromContext(req.Context())

	slog.Info("Handling PyPI simple package request", "registry", reg.Name, "repository", repo.Name, "package", pkgName, "type", repo.Type)

	var allFiles []templateFile
	var err error

	switch repo.Type {
	case domain.RepositoryTypeVirtual:
		allFiles, err = p.buildVirtualPackageFiles(req, reg, repo, pkgName, normalized)
	case domain.RepositoryTypeProxy:
		allFiles, err = p.proxySimplePackage(req, reg, repo, pkgName, normalized)
	default:
		allFiles, err = p.fetchLocalPackageFiles(req, reg, repo, normalized)
	}

	if err != nil {
		slog.Error("Failed to fetch package files", "error", err, "package", pkgName, "repository", repo.Name)
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "package not found" {
			http.Error(w, "package not found", http.StatusNotFound)
		} else if repo.Type == domain.RepositoryTypeProxy && err.Error() == "bad gateway" {
			http.Error(w, "bad gateway", http.StatusBadGateway)
		} else {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	if acceptsJSON(req) {
		p.serveSimplePackageJSON(w, pkgName, allFiles)
		return
	}

	p.serveSimplePackageHTML(w, reg, repo, pkgName, allFiles)
}

func (p *PyPIProtocol) buildVirtualPackageFiles(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName, normalized string) ([]templateFile, error) {
	var allFiles []templateFile
	seen := make(map[string]bool)

	for _, member := range repo.VirtualMembers {
		var tFiles []templateFile
		var err error

		if member.MemberRepo.Type == domain.RepositoryTypeProxy {
			tFiles, err = p.proxySimplePackage(req, reg, &member.MemberRepo, pkgName, normalized)
		} else {
			tFiles, err = p.fetchLocalPackageFiles(req, reg, &member.MemberRepo, normalized)
		}

		if err == nil {
			for _, tf := range tFiles {
				if !seen[tf.Filename] {
					seen[tf.Filename] = true
					allFiles = append(allFiles, tf)
				}
			}
		} else {
			slog.Warn("Failed to fetch files from virtual member", "member", member.MemberRepo.Name, "package", pkgName, "error", err)
		}
	}

	if len(allFiles) == 0 {
		return nil, errors.New("package not found")
	}
	return allFiles, nil
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
		slog.Error("Failed to encode simple index JSON", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) serveSimplePackageJSON(w http.ResponseWriter, pkgName string, tFiles []templateFile) {
	w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
	resp := SimplePackageResponse{
		Meta:  SimpleMeta{APIVersion: "1.1"},
		Name:  pkgName,
		Files: []SimpleFile{},
	}

	versionsSet := make(map[string]struct{})
	for _, tf := range tFiles {
		if tf.Version != "" && tf.Version != "unknown" {
			if _, exists := versionsSet[tf.Version]; !exists {
				versionsSet[tf.Version] = struct{}{}
				resp.Versions = append(resp.Versions, tf.Version)
			}
		}

		resp.Files = append(resp.Files, SimpleFile{
			Filename:   tf.Filename,
			URL:        tf.URL,
			Hashes:     SimpleFileHashes{SHA256: tf.Hash},
			Size:       tf.Size,
			UploadTime: tf.UploadTime,
		})
	}

	if resp.Versions == nil {
		resp.Versions = []string{}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode simple package JSON", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) serveSimplePackageHTML(w http.ResponseWriter, reg *domain.Registry, repo *domain.Repository, pkgName string, files []templateFile) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		RepoName     string
		RegistryName string
		PackageName  string
		Files        []templateFile
	}{
		RepoName:     repo.Name,
		RegistryName: reg.Name,
		PackageName:  pkgName,
		Files:        files,
	}

	if err := packageTemplate.ExecuteTemplate(w, "package.html", data); err != nil {
		slog.Error("Failed to execute package template", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) handleDownload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	filename := chi.URLParam(req, "filename")

	slog.Info("Handling PyPI file download", "repository", repo.Name, "filename", filename, "type", repo.Type)

	if repo.Type == domain.RepositoryTypeVirtual {
		registry.DispatchVirtualDownload(w, req, repo, p.MountRoutes())
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if repo.Type == domain.RepositoryTypeProxy {
				slog.Info("File not in local DB, delegating to proxyDownload", "filename", filename)
				p.proxyDownload(w, req, repo, filename)
				return
			}
			slog.Warn("File not found in local repository", "filename", filename)
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		slog.Error("Database error checking for file", "error", err, "filename", filename)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	p.serveFileFromStorage(w, req, pkgFile)
}

func (p *PyPIProtocol) serveFileFromStorage(w http.ResponseWriter, req *http.Request, pkgFile *domain.PackageFile) {
	slog.Info("Serving file from storage", "filename", pkgFile.Filename, "blobKey", pkgFile.BlobKey)
	reader, err := p.storage.Get(req.Context(), pkgFile.BlobKey)
	if err != nil {
		slog.Error("Failed to read file from storage", "error", err, "blobKey", pkgFile.BlobKey)
		http.Error(w, "failed to read file from storage", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", pkgFile.Filename))
	io.Copy(w, reader)
}

func (p *PyPIProtocol) proxySimplePackage(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName, normalized string) ([]templateFile, error) {
	upstreamURL := fmt.Sprintf("%s/simple/%s/", strings.TrimSuffix(repo.UpstreamURL, "/"), pkgName)
	cacheKey := fmt.Sprintf("metadata/%d/simple/%s?json=true", repo.ID, pkgName)

	slog.Info("Fetching simple package from upstream", "upstreamURL", upstreamURL)

	headers := map[string]string{
		"Accept": "application/vnd.pypi.simple.v1+json",
	}

	content, err := p.cacheService.FetchMetadata(req.Context(), cacheKey, upstreamURL, headers)
	if err != nil {
		slog.Error("Failed to fetch upstream metadata", "error", err, "upstreamURL", upstreamURL)
		return nil, fmt.Errorf("bad gateway")
	}

	var upstreamResp SimplePackageResponse
	if err := json.Unmarshal(content, &upstreamResp); err != nil {
		slog.Error("Failed to unmarshal upstream JSON", "error", err, "upstreamURL", upstreamURL)
		return nil, fmt.Errorf("bad upstream response")
	}

	var tFiles []templateFile
	for _, f := range upstreamResp.Files {
		encodedUpstream := ""
		if f.URL != "" {
			encodedUpstream = fmt.Sprintf("?pkg=%s&upstream=%s", url.QueryEscape(pkgName), url.QueryEscape(f.URL))
		}

		finalURL := fmt.Sprintf("/registries/%s/repositories/%s/packages/%s%s", reg.Name, repo.Name, f.Filename, encodedUpstream)

		tFiles = append(tFiles, templateFile{
			Filename:   f.Filename,
			Hash:       f.Hashes.SHA256,
			URL:        finalURL,
			Size:       f.Size,
			UploadTime: f.UploadTime,
			Version:    extractVersion(f.Filename, pkgName),
		})
	}

	return tFiles, nil
}


func (p *PyPIProtocol) proxyDownload(w http.ResponseWriter, req *http.Request, repo *domain.Repository, filename string) {
	pkgName := req.URL.Query().Get("pkg")
	upstreamURL := req.URL.Query().Get("upstream")

	if pkgName == "" || upstreamURL == "" {
		slog.Warn("Missing proxy parameters", "pkg", pkgName, "upstream", upstreamURL)
		http.Error(w, "missing proxy parameters: pkg or upstream", http.StatusBadRequest)
		return
	}

	normalized := NormalizeName(pkgName)
	blobKey := fmt.Sprintf("%d/%s", repo.ID, filename)

	slog.Info("Initiating proxy download", "filename", filename, "upstream", upstreamURL)

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, pkgName, normalized)
	if err != nil {
		slog.Error("Failed to get or create package for proxy download", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tw := &trackedWriter{ResponseWriter: w}

	_, err, _ = p.downloadSF.Do(blobKey, func() (interface{}, error) {
		// Verify again inside SF
		_, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
		if err == nil {
			slog.Info("File downloaded concurrently by another request, skipping stream", "filename", filename)
			return nil, nil // Already downloaded
		}

		slog.Info("Streaming and saving file from upstream", "filename", filename)
		err = p.cacheService.StreamAndSave(req.Context(), tw, upstreamURL, func(r io.Reader, size int64) error {
			return p.saveProxiedFile(req.Context(), r, size, repo.ID, pkg, filename, blobKey, pkgName)
		})
		return nil, err
	})

	if err != nil {
		slog.Error("Error during proxy download singleflight", "error", err)
		return
	}

	if !tw.written {
		// We were a waiter, or it was already in DB. Serve from storage.
		slog.Info("Serving proxied file from storage after wait", "filename", filename)
		pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
		if err != nil {
			slog.Error("Failed to retrieve file from DB after proxy download", "error", err)
			http.Error(w, "internal error retrieving downloaded file", http.StatusInternalServerError)
			return
		}
		p.serveFileFromStorage(w, req, pkgFile)
	}
}

func (p *PyPIProtocol) saveProxiedFile(ctx context.Context, r io.Reader, size int64, repoID uint, pkg *domain.Package, filename, blobKey, pkgName string) error {
	hasher := sha256.New()
	tee := io.TeeReader(r, hasher)

	if err := p.storage.Put(ctx, blobKey, tee); err != nil {
		return err
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))
	version := extractVersion(filename, pkgName)

	pkgFile := &domain.PackageFile{
		PackageID: pkg.ID,
		Version:   version,
		Filename:  filename,
		Hash:      hashString,
		Size:      size,
		BlobKey:   blobKey,
	}

	return p.packageFileStore.Create(pkgFile)
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
			// If it failed due to a unique constraint violation from a concurrent request, try to fetch it again.
			if existingPkg, errGet := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID); errGet == nil {
				return existingPkg, nil
			}
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
	slog.Info("Handling PyPI package upload", "repository", repo.Name)

	if repo.Type == domain.RepositoryTypeVirtual {
		slog.Warn("Attempted upload to virtual repository")
		http.Error(w, "cannot upload to virtual repository", http.StatusBadRequest)
		return
	}

	meta, err := parseUploadForm(req)
	if err != nil {
		slog.Warn("Failed to parse upload form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer meta.File.Close()

	slog.Info("Parsed upload form", "package", meta.Name, "version", meta.Version, "filename", meta.Filename)

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, meta.Name, meta.NormalizedName)
	if err != nil {
		slog.Error("Failed to get or create package during upload", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = p.packageFileStore.GetByFilenameAndPackage(meta.Filename, pkg.ID)
	if err == nil {
		slog.Warn("File already exists", "filename", meta.Filename)
		http.Error(w, "file already exists", http.StatusConflict)
		return
	}

	if err := p.storeFileAndMetadata(req.Context(), repo.ID, pkg, meta); err != nil {
		slog.Error("Failed to store file and metadata", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Successfully uploaded PyPI package", "filename", meta.Filename, "repository", repo.Name)
	w.WriteHeader(http.StatusOK)
}
