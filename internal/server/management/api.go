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

	return r
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
