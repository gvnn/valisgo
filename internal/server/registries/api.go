package registries

import (
	"context"
	"log/slog"
	"net/http"
	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/registry"
	"valisgo/internal/registry/file"
	"valisgo/internal/registry/npm"
	"valisgo/internal/registry/pypi"
	"valisgo/internal/storage"
	"valisgo/internal/store"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type API struct {
	registryStore   domain.RegistryStore
	repositoryStore domain.RepositoryStore

	protocolHandlers map[domain.RegistryFormat]chi.Router
}

func NewAPI(db *gorm.DB, storage storage.Storage) *API {
	api := &API{
		registryStore:    store.NewRegistryStore(db),
		repositoryStore:  store.NewRepositoryStore(db),
		protocolHandlers: make(map[domain.RegistryFormat]chi.Router),
	}

	packageStore := store.NewPackageStore(db)
	packageFileStore := store.NewPackageFileStore(db)

	slog.Info("Registering protocol handlers")
	api.RegisterProtocolHandler(domain.FormatPyPI, pypi.NewPyPIProtocol(packageStore, packageFileStore, storage, proxy.NewCacheService(storage)))
	api.RegisterProtocolHandler(domain.FormatNPM, npm.NewNPMProtocol(packageStore, packageFileStore, storage, proxy.NewCacheService(storage)))
	api.RegisterProtocolHandler(domain.FormatFile, file.NewFileProtocol(packageStore, packageFileStore, storage))

	return api
}

func (a *API) RegisterProtocolHandler(format domain.RegistryFormat, p registry.Protocol) {
	a.protocolHandlers[format] = p.MountRoutes()
}

func (a *API) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{registryName}/repositories/{repositoryName}", func(r chi.Router) {

		// When a path is called, this middleware does the DB checks
		r.Use(a.databaseValidationMiddleware)

		r.Handle("/*", http.HandlerFunc(a.dispatchToProtocol))
	})

	return r
}

func (a *API) databaseValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		regName := chi.URLParam(req, "registryName")
		repoName := chi.URLParam(req, "repositoryName")

		reg, err := a.registryStore.GetByName(regName)
		if err != nil {
			http.Error(w, "registry not found or format mismatch", http.StatusNotFound)
			return
		}

		repo, err := a.repositoryStore.GetByNameAndRegistryID(repoName, reg.ID)
		if err != nil {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		ctx := context.WithValue(req.Context(), domain.RegistryCtxKey, reg)
		ctx = context.WithValue(ctx, domain.RepoCtxKey, repo)

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (a *API) dispatchToProtocol(w http.ResponseWriter, req *http.Request) {
	reg := domain.RegistryFromContext(req.Context())

	protoRouter, ok := a.protocolHandlers[reg.Format]
	if !ok {
		http.Error(w, "protocol handler not configured for this format", http.StatusNotImplemented)
		return
	}

	remainingPath := chi.URLParam(req, "*")
	req.URL.Path = "/" + remainingPath

	// Handoff to PyPI / Go / NPM / File
	protoRouter.ServeHTTP(w, req)
}
