package npm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"


	"valisgo/internal/domain"
	"valisgo/internal/registry"
)

func (p *NPMProtocol) getOrCreatePackage(ctx context.Context, repoID uint, name, normalizedName string) (*domain.Package, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID)
	if err != nil {
		return nil, err
	}
	if pkg != nil {
		return pkg, nil
	}

	newPkg := &domain.Package{
		Name:           name,
		NormalizedName: normalizedName,
		RepositoryID:   repoID,
	}

	err = p.packageStore.Create(newPkg)
	if err == nil {
		return newPkg, nil
	}

	existingPkg, errGet := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID)
	if errGet == nil && existingPkg != nil {
		return existingPkg, nil
	}

	return nil, err
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
	if registry.HandleInternalError(w, err) {
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
