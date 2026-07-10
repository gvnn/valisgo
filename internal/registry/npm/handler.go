package npm

import (
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
)

type NPMProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
	cacheService     *proxy.CacheService
	downloadSF       singleflight.Group
}

func NewNPMProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage, cacheService *proxy.CacheService) *NPMProtocol {
	return &NPMProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
		cacheService:     cacheService,
	}
}

func (p *NPMProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/{package}", p.handleMetadata)
	r.Get("/@{scope}/{package}", p.handleMetadata)
	r.Get("/{package}/-/{filename}", p.handleDownload)
	r.Get("/@{scope}/{package}/-/{filename}", p.handleDownload)

	r.Put("/{package}", p.handleUpload)
	r.Put("/@{scope}/{package}", p.handleUpload)

	return r
}

type trackedWriter struct {
	http.ResponseWriter
	written bool
}

func (tw *trackedWriter) Write(b []byte) (int, error) {
	tw.written = true
	return tw.ResponseWriter.Write(b)
}

func (tw *trackedWriter) WriteHeader(statusCode int) {
	tw.written = true
	tw.ResponseWriter.WriteHeader(statusCode)
}
