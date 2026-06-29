package registry

import (
	"github.com/go-chi/chi/v5"
)

type Protocol interface {
	MountRoutes() chi.Router
}
