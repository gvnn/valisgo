package npm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/registry"

	"github.com/go-chi/chi/v5"
)

func (p *NPMProtocol) handleDownload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	filename := chi.URLParam(req, "filename")

	slog.Info("Handling NPM file download", "repository", repo.Name, "filename", filename, "type", repo.Type)

	if repo.Type == domain.RepositoryTypeVirtual {
		registry.DispatchVirtualDownload(w, req, repo, p.MountRoutes())
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
	if registry.HandleInternalError(w, err) {
		return
	}
	if pkgFile != nil {
		p.serveFileFromStorage(w, req, pkgFile)
		return
	}

	if repo.Type == domain.RepositoryTypeProxy {
		p.proxyDownload(w, req, repo, filename)
		return
	}
	http.Error(w, "file not found", http.StatusNotFound)
}

func (p *NPMProtocol) proxyDownload(w http.ResponseWriter, req *http.Request, repo *domain.Repository, filename string) {
	pkgName := req.URL.Query().Get("pkg")
	upstreamURL := req.URL.Query().Get("upstream")

	if pkgName == "" || upstreamURL == "" {
		http.Error(w, "missing proxy parameters", http.StatusBadRequest)
		return
	}

	blobKey := fmt.Sprintf("%d/%s", repo.ID, filename)

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, pkgName, pkgName)
	if registry.HandleInternalError(w, err) {
		return
	}

	tw := &trackedWriter{ResponseWriter: w}

	_, err, _ = p.downloadSF.Do(blobKey, func() (interface{}, error) {
		pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
		if err == nil && pkgFile != nil {
			return nil, nil
		}

		err = p.cacheService.StreamAndSave(req.Context(), tw, upstreamURL, func(r io.Reader, size int64) error {
			return p.saveProxiedFile(req.Context(), r, size, repo.ID, pkg, filename, blobKey)
		})
		return nil, err
	})

	if err != nil {
		slog.Error("Error during proxy download", "error", err)
		return
	}

	if tw.written {
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
	if registry.HandleInternalError(w, err) {
		return
	}
	p.serveFileFromStorage(w, req, pkgFile)
}

func (p *NPMProtocol) saveProxiedFile(ctx context.Context, r io.Reader, size int64, repoID uint, pkg *domain.Package, filename, blobKey string) error {
	hasher := sha256.New()
	tee := io.TeeReader(r, hasher)

	if err := p.storage.Put(ctx, blobKey, tee); err != nil {
		return err
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))

	pkgFile := &domain.PackageFile{
		PackageID: pkg.ID,
		Version:   extractVersion(filename, pkg.Name),
		Filename:  filename,
		Hash:      hashString,
		Size:      size,
		BlobKey:   blobKey,
	}

	return p.packageFileStore.Create(pkgFile)
}

func (p *NPMProtocol) serveFileFromStorage(w http.ResponseWriter, req *http.Request, pkgFile *domain.PackageFile) {
	reader, err := p.storage.Get(req.Context(), pkgFile.BlobKey)
	if err != nil {
		http.Error(w, "failed to read file from storage", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, reader)
}
