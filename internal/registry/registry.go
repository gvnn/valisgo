package registry

import (
	"valisgo/internal/domain"

	"github.com/go-chi/chi/v5"
)

type Protocol interface {
	MountRoutes(repo *domain.Repository) chi.Router
}
