package pypi

import (
	"embed"
	"html/template"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
)

type PyPIProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
	cacheService     *proxy.CacheService
	downloadSF       singleflight.Group
}

//go:embed templates/*.html
var templateFS embed.FS

var (
	indexTemplate   = template.Must(template.ParseFS(templateFS, "templates/index.html"))
	packageTemplate = template.Must(template.ParseFS(templateFS, "templates/package.html"))
)

func NewPyPIProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage, cacheService *proxy.CacheService) *PyPIProtocol {
	return &PyPIProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
		cacheService:     cacheService,
	}
}

func (p *PyPIProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/simple/", p.handleSimpleIndex)
	r.Get("/simple/{package}/", p.handleSimplePackage)
	r.Get("/packages/{filename}", p.handleDownload)
	r.Post("/", p.handleUpload)

	return r
}


