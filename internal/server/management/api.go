package management

import (
	"encoding/json"
	"errors"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/store"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type API struct {
	db              *gorm.DB
	registryStore   domain.RegistryStore
	repositoryStore domain.RepositoryStore
}

func NewAPI(db *gorm.DB) *API {
	return &API{
		db:              db,
		registryStore:   store.NewRegistryStore(db),
		repositoryStore: store.NewRepositoryStore(db),
	}
}

func (a *API) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/openapi.yaml")
	})

	r.Get("/registries", a.listRegistries)
	r.Post("/registries", a.createRegistry)

	r.Get("/repositories", a.listRepositories)
	r.Post("/repositories", a.createRepository)

	return r
}

type repositoryInput struct {
	Name         string                `json:"Name"`
	RegistryName string                `json:"RegistryName"`
	Type         domain.RepositoryType `json:"Type"`
	UpstreamURL  string                `json:"UpstreamURL"`
}

func (a *API) processCreateRegistry(registry *domain.Registry) (int, error) {
	existing, err := a.registryStore.GetByName(registry.Name)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if existing != nil {
		return http.StatusConflict, errors.New("registry already exists")
	}

	if err := a.registryStore.Create(registry); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusCreated, nil
}

func (a *API) processCreateRepository(input *repositoryInput) (*domain.Repository, int, error) {
	registry, err := a.registryStore.GetByName(input.RegistryName)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if registry == nil {
		return nil, http.StatusNotFound, errors.New("registry not found")
	}

	existing, err := a.repositoryStore.GetByNameAndRegistryID(input.Name, registry.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if existing != nil {
		return nil, http.StatusConflict, errors.New("repository already exists in this registry")
	}

	repo := &domain.Repository{
		Name:        input.Name,
		RegistryID:  registry.ID,
		Type:        input.Type,
		UpstreamURL: input.UpstreamURL,
	}

	if err := a.repositoryStore.Create(repo); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return repo, http.StatusCreated, nil
}

func (a *API) processListRepositories(registryName string) ([]*domain.Repository, int, error) {
	if registryName == "" {
		repos, err := a.repositoryStore.All()
		return repos, http.StatusInternalServerError, err
	}

	registry, err := a.registryStore.GetByName(registryName)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if registry == nil {
		return nil, http.StatusNotFound, errors.New("registry not found")
	}

	repos, err := a.repositoryStore.ListByRegistryID(registry.ID)
	return repos, http.StatusInternalServerError, err
}

func (a *API) createRegistry(w http.ResponseWriter, r *http.Request) {
	var registry domain.Registry
	if err := json.NewDecoder(r.Body).Decode(&registry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := a.processCreateRegistry(&registry)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&registry)
}

func (a *API) listRegistries(w http.ResponseWriter, r *http.Request) {
	registries, err := a.registryStore.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registries)
}

func (a *API) createRepository(w http.ResponseWriter, r *http.Request) {
	var input repositoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, status, err := a.processCreateRepository(&input)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(repo)
}

func (a *API) listRepositories(w http.ResponseWriter, r *http.Request) {
	registryName := r.URL.Query().Get("registry")

	repos, status, err := a.processListRepositories(registryName)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}
