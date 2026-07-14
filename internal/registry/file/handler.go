package file

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"valisgo/internal/domain"
	"valisgo/internal/registry"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
)

type FileProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
}

func NewFileProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage) *FileProtocol {
	return &FileProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
	}
}

func (p *FileProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/*", p.handleDownload)
	r.Put("/*", p.handleUpload)
	r.Post("/*", p.handleUpload)

	return r
}

func (p *FileProtocol) handleDownload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())

	filePath := chi.URLParam(req, "*")
	if filePath == "" {
		http.Error(w, "bad request: empty path", http.StatusBadRequest)
		return
	}

	if repo.Type == domain.RepositoryTypeVirtual {
		registry.DispatchVirtualDownload(w, req, repo, p.MountRoutes())
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filePath, repo.ID)

	if registry.HandleInternalError(w, err) {
		return
	}

	if pkgFile == nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	reader, err := p.storage.Get(req.Context(), pkgFile.BlobKey)

	if err != nil {
		http.Error(w, "failed to read file from storage", http.StatusInternalServerError)
		return
	}

	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", pkgFile.Size))
	io.Copy(w, reader)
}

func (p *FileProtocol) getOrCreateRootPackage(ctx context.Context, repoID uint) (*domain.Package, error) {
	pkgName := "root"
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(pkgName, repoID)

	if err != nil {
		return nil, fmt.Errorf("internal error fetching root package: %w", err)
	}

	if pkg != nil {
		return pkg, nil
	}

	newPkg := &domain.Package{
		Name:           pkgName,
		NormalizedName: pkgName,
		RepositoryID:   repoID,
	}

	err = p.packageStore.Create(newPkg)

	if err == nil {
		return newPkg, nil
	}

	existing, errGet := p.packageStore.GetByNormalizedNameAndRepository(pkgName, repoID)

	if errGet == nil && existing != nil {
		return existing, nil
	}

	return nil, fmt.Errorf("failed to create root package: %w", err)
}

type countTeeReader struct {
	r io.Reader
	w io.Writer
	n int64
}

func (c *countTeeReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.n += int64(n)
		if _, wErr := c.w.Write(p[:n]); wErr != nil {
			return n, wErr
		}
	}
	return n, err
}

func extractPayload(req *http.Request, filePath string) (io.ReadCloser, string, error) {
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return req.Body, filePath, nil
	}

	if err := req.ParseMultipartForm(10 << 20); err != nil {
		return nil, "", fmt.Errorf("invalid form: %w", err)
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		file, header, err = req.FormFile("content")
		if err != nil {
			return nil, "", errors.New("missing file or content field")
		}
	}

	if strings.HasSuffix(filePath, "/") {
		filePath += header.Filename
	}
	return file, filePath, nil
}

func (p *FileProtocol) storeFileAndMetadata(ctx context.Context, repoID, pkgID uint, filePath string, reader io.Reader) error {
	hasher := sha256.New()
	ctr := &countTeeReader{
		r: reader,
		w: hasher,
	}

	blobKey := fmt.Sprintf("%d/%s", repoID, filePath)

	if err := p.storage.Put(ctx, blobKey, ctr); err != nil {
		return fmt.Errorf("failed to store file: %w", err)
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))

	pkgFile := &domain.PackageFile{
		PackageID: pkgID,
		Version:   "latest",
		Filename:  filePath,
		Hash:      hashString,
		Size:      ctr.n,
		BlobKey:   blobKey,
	}

	if err := p.packageFileStore.Create(pkgFile); err != nil {
		_ = p.storage.Delete(ctx, blobKey)
		return fmt.Errorf("failed to save file metadata: %w", err)
	}

	return nil
}

func (p *FileProtocol) handleUpload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())

	filePath := chi.URLParam(req, "*")
	if filePath == "" {
		http.Error(w, "bad request: empty path", http.StatusBadRequest)
		return
	}

	if repo.Type == domain.RepositoryTypeVirtual {
		http.Error(w, "cannot upload to virtual repository", http.StatusBadRequest)
		return
	}

	reader, finalFilePath, err := extractPayload(req, filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if reader != req.Body {
		defer reader.Close()
	}

	pkg, err := p.getOrCreateRootPackage(req.Context(), repo.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndPackage(finalFilePath, pkg.ID)
	if registry.HandleInternalError(w, err) {
		return
	}
	if pkgFile != nil {
		http.Error(w, "file already exists", http.StatusConflict)
		return
	}

	if err := p.storeFileAndMetadata(req.Context(), repo.ID, pkg.ID, finalFilePath, reader); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
