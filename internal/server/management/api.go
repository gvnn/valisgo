package management

import (
	"encoding/json"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/store"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type API struct {
	db            *gorm.DB
	registryStore domain.RegistryStore
}

func NewAPI(db *gorm.DB) *API {
	return &API{
		db:            db,
		registryStore: store.NewRegistryStore(db),
	}
}

func (a *API) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/registries", a.listRegistries)
	r.Post("/registries", a.createRegistry)

	return r
}

func (a *API) createRegistry(w http.ResponseWriter, r *http.Request) {
	var registry domain.Registry
	if err := json.NewDecoder(r.Body).Decode(&registry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing, err := a.registryStore.GetByName(registry.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "registry already exists", http.StatusConflict)
		return
	}

	if err := a.registryStore.Create(&registry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
