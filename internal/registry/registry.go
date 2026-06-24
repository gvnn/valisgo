package registry

import (
	"github.com/go-chi/chi/v5"
)

type Repository struct {
	Name   string
	Format string
}

type Protocol interface {
	MountRoutes(repo *Repository) chi.Router
}
