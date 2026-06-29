package pypi

import (
	"fmt"
	"net/http"
	"valisgo/internal/domain"

	"github.com/go-chi/chi/v5"
)

type PyPIProtocol struct{}

func (p *PyPIProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/simple/{package}", func(w http.ResponseWriter, req *http.Request) {
		repo := domain.RepositoryFromContext(req.Context())
		pkgName := chi.URLParam(req, "package")

		response := fmt.Sprintf("hosted pypi metadata for package '%s' in repository '%s'", pkgName, repo.Name)
		w.Write([]byte(response))
	})

	r.Get("/packages/*", func(w http.ResponseWriter, req *http.Request) {
		repo := domain.RepositoryFromContext(req.Context())

		response := fmt.Sprintf("Downloading wheel from repository: %s", repo.Name)
		w.Write([]byte(response))
	})

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		repo := domain.RepositoryFromContext(req.Context())

		fmt.Printf("Upload triggered for %s\n", repo.Name)
		w.WriteHeader(http.StatusOK)
	})

	return r
}
