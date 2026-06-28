package pypi

import (
	"net/http"

	"valisgo/internal/domain"

	"github.com/go-chi/chi/v5"
)

type PyPIProtocol struct{}

func (p *PyPIProtocol) MountRoutes(repo *domain.Repository) chi.Router {
	r := chi.NewRouter()

	r.Get("/simple/{package}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hosted pypi metadata"))
	})

	r.Get("/packages/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hosted pypi wheel"))
	})

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return r
}
