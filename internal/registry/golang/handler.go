package golang

import (
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
)

func NewGoProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage, cacheService *proxy.CacheService) *GoProtocol {
	return &GoProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
		cacheService:     cacheService,
	}
}

func (p *GoProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	// Catch-all route for Go module paths since they can contain slashes
	r.Get("/*", p.handleCatchAll)
	r.Put("/*", p.handleUpload)

	return r
}

func (p *GoProtocol) handleCatchAll(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")

	modulePath, version, ext, err := ParsePath(path)
	if err != nil {
		if err == ErrInvalidPath {
			http.Error(w, "invalid goproxy path", http.StatusBadRequest)
		} else {
			http.Error(w, "unknown action", http.StatusNotFound)
		}
		return
	}

	switch ext {
	case "list":
		p.handleListVersions(w, r, modulePath)
	case ".info":
		p.handleVersionInfo(w, r, modulePath, version)
	case ".mod", ".zip":
		p.handleDownload(w, r, modulePath, version, ext)
	}
}
