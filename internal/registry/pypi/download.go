package pypi

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
		slog.Error("Database error checking for file", "error", err, "filename", filename)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if pkgFile != nil {
		p.serveFileFromStorage(w, req, pkgFile)
		return
	}

	if repo.Type == domain.RepositoryTypeProxy {
		slog.Info("File not in local DB, delegating to proxyDownload", "filename", filename)
		p.proxyDownload(w, req, repo, filename)
		return
	}

	slog.Warn("File not found in local repository", "filename", filename)
	http.Error(w, "file not found", http.StatusNotFound)
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
		pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
		if err == nil && pkgFile != nil {
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

	if tw.written {
		return
	}

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
