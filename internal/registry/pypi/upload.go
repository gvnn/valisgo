package pypi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"valisgo/internal/domain"

	"gorm.io/gorm"
)





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
