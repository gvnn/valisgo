package npm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"


	"valisgo/internal/domain"

	"gorm.io/gorm"
)

func (p *NPMProtocol) getOrCreatePackage(ctx context.Context, repoID uint, name, normalizedName string) (*domain.Package, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		pkg = &domain.Package{
			Name:           name,
			NormalizedName: normalizedName,
			RepositoryID:   repoID,
		}
		if err := p.packageStore.Create(pkg); err != nil {
			if existingPkg, errGet := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID); errGet == nil {
				return existingPkg, nil
			}
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return pkg, nil
}

func (p *NPMProtocol) handleUpload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	slog.Info("Handling NPM package upload", "repository", repo.Name)

	if repo.Type == domain.RepositoryTypeVirtual {
		http.Error(w, "cannot upload to virtual repository", http.StatusBadRequest)
		return
	}

	var payload PublishPayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		slog.Warn("Failed to parse NPM publish payload", "error", err)
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	pkgName := payload.Name
	normalized := pkgName // For NPM we can just use the name as normalized

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, pkgName, normalized)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for filename, attachment := range payload.Attachments {
		data, err := base64.StdEncoding.DecodeString(attachment.Data)
		if err != nil {
			slog.Warn("Failed to decode attachment data", "filename", filename)
			continue
		}

		// Save to storage
		blobKey := fmt.Sprintf("%d/%s", repo.ID, filename)
		reader := bytes.NewReader(data)
		hasher := sha256.New()
		tee := io.TeeReader(reader, hasher)

		if err := p.storage.Put(req.Context(), blobKey, tee); err != nil {
			slog.Error("Failed to store file", "filename", filename, "error", err)
			continue
		}

		hashString := fmt.Sprintf("%x", hasher.Sum(nil))

		version := extractVersion(filename, pkgName)

		pkgFile := &domain.PackageFile{
			PackageID: pkg.ID,
			Version:   version,
			Filename:  filename,
			Hash:      hashString,
			Size:      int64(len(data)),
			BlobKey:   blobKey,
		}

		if err := p.packageFileStore.Create(pkgFile); err != nil {
			slog.Error("Failed to save file metadata", "error", err)
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"success": true}`))
}
