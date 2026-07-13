package golang

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"valisgo/internal/domain"
	"valisgo/internal/registry"

	"gorm.io/gorm"
)

func (p *GoProtocol) handleDownload(w http.ResponseWriter, req *http.Request, modulePath, version, ext string) {
	repo := domain.RepositoryFromContext(req.Context())
	filename := fmt.Sprintf("%s%s", version, ext)

	slog.Info("Handling Go file download", "repository", repo.Name, "module", modulePath, "version", version, "ext", ext, "type", repo.Type)

	if repo.Type == domain.RepositoryTypeVirtual {
		registry.DispatchVirtualDownload(w, req, repo, p.MountRoutes())
		return
	}

	// Resolve the package first so file lookups are scoped to this module.
	// Go filenames are just "<version>.<ext>" and collide across modules
	// that share a version, so a repository-wide lookup is not safe here.
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(modulePath, repo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if repo.Type == domain.RepositoryTypeProxy {
				p.proxyDownload(w, req, repo, modulePath, version, filename)
				return
			}
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndPackage(filename, pkg.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if repo.Type == domain.RepositoryTypeProxy {
				p.proxyDownload(w, req, repo, modulePath, version, filename)
				return
			}
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	p.serveFileFromStorage(w, req, pkgFile)
}

func (p *GoProtocol) proxyDownload(w http.ResponseWriter, req *http.Request, repo *domain.Repository, modulePath, version, filename string) {
	upstreamURL := fmt.Sprintf("%s/%s/@v/%s", strings.TrimSuffix(repo.UpstreamURL, "/"), modulePath, filename)
	blobKey := goBlobKey(repo.ID, modulePath, filename)

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, modulePath)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tw := &trackedWriter{ResponseWriter: w}

	_, err, _ = p.downloadSF.Do(blobKey, func() (interface{}, error) {
		_, err := p.packageFileStore.GetByFilenameAndPackage(filename, pkg.ID)
		if err == nil {
			return nil, nil // Already downloaded by another request
		}

		err = p.cacheService.StreamAndSave(req.Context(), tw, upstreamURL, func(r io.Reader, size int64) error {
			return p.saveProxiedFile(req.Context(), r, size, repo.ID, pkg, filename, version, blobKey)
		})
		return nil, err
	})

	if err != nil {
		slog.Error("Error during proxy download", "error", err)
		http.Error(w, "not found upstream", http.StatusNotFound)
		return
	}

	if !tw.written {
		pkgFile, err := p.packageFileStore.GetByFilenameAndPackage(filename, pkg.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		p.serveFileFromStorage(w, req, pkgFile)
	}
}

// goBlobKey builds a storage key scoped to the module so that different
// modules sharing a version (e.g. two modules both at v0.3.0) don't collide
// on the same "<version>.<ext>" filename.
func goBlobKey(repoID uint, modulePath, filename string) string {
	return fmt.Sprintf("%d/%s/%s", repoID, modulePath, filename)
}

func (p *GoProtocol) saveProxiedFile(ctx context.Context, r io.Reader, size int64, repoID uint, pkg *domain.Package, filename, version, blobKey string) error {
	hasher := sha256.New()
	tee := io.TeeReader(r, hasher)

	if err := p.storage.Put(ctx, blobKey, tee); err != nil {
		return err
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))

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

func (p *GoProtocol) serveFileFromStorage(w http.ResponseWriter, req *http.Request, pkgFile *domain.PackageFile) {
	reader, err := p.storage.Get(req.Context(), pkgFile.BlobKey)
	if err != nil {
		http.Error(w, "failed to read file from storage", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	if strings.HasSuffix(pkgFile.Filename, ".mod") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else if strings.HasSuffix(pkgFile.Filename, ".zip") {
		w.Header().Set("Content-Type", "application/zip")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	
	io.Copy(w, reader)
}

func (p *GoProtocol) getOrCreatePackage(ctx context.Context, repoID uint, modulePath string) (*domain.Package, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(modulePath, repoID)
	if err == nil {
		return pkg, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	newPkg := &domain.Package{
		Name:           modulePath,
		NormalizedName: modulePath,
		RepositoryID:   repoID,
	}

	if err := p.packageStore.Create(newPkg); err != nil {
		// Try fetching again in case of race condition
		return p.packageStore.GetByNormalizedNameAndRepository(modulePath, repoID)
	}

	return newPkg, nil
}
