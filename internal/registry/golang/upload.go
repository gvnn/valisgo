package golang

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/registry"

	"github.com/go-chi/chi/v5"
)

func (p *GoProtocol) handleUpload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())

	if repo.Type == domain.RepositoryTypeVirtual {
		http.Error(w, "cannot upload to virtual repository", http.StatusBadRequest)
		return
	}
	if repo.Type == domain.RepositoryTypeProxy {
		http.Error(w, "cannot upload to proxy repository", http.StatusBadRequest)
		return
	}

	path := chi.URLParam(req, "*")

	modulePath, version, ext, err := ParsePath(path)

	if err == ErrInvalidPath {
		http.Error(w, "invalid goproxy path for upload", http.StatusBadRequest)
		return
	}

	if err != nil || ext == "list" {
		http.Error(w, "unsupported upload type", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("%s%s", version, ext)
	slog.Info("Handling Go module file upload", "repository", repo.Name, "module", modulePath, "version", version, "file", filename)

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, modulePath)

	if registry.HandleInternalError(w, err) {
		return
	}

	blobKey := goBlobKey(repo.ID, modulePath, filename)

	hasher := sha256.New()

	// Create a proxy reader that wraps the request body to count bytes read
	lr := &lengthReader{r: req.Body}
	tee := io.TeeReader(lr, hasher)

	if err := p.storage.Put(req.Context(), blobKey, tee); err != nil {
		slog.Error("Failed to store uploaded file", "filename", filename, "error", err)
		http.Error(w, "failed to store file", http.StatusInternalServerError)
		return
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))

	pkgFile := &domain.PackageFile{
		PackageID: pkg.ID,
		Version:   version,
		Filename:  filename,
		Hash:      hashString,
		Size:      lr.size,
		BlobKey:   blobKey,
	}

	if err := p.packageFileStore.Create(pkgFile); err != nil {
		slog.Error("Failed to save file metadata", "error", err)
		http.Error(w, "failed to save metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"success": true}`))
}
