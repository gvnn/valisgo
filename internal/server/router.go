package server

import (
	"valisgo/internal/auth"

	"github.com/casbin/casbin/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	enforcer      *casbin.Enforcer
	authenticator *auth.Authenticator
}

func NewServer(enforcer *casbin.Enforcer, authenticator *auth.Authenticator) *Server {
	return &Server{
		enforcer:      enforcer,
		authenticator: authenticator,
	}
}

func (s *Server) SetupRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if s.enforcer != nil {
		r.Use(CasbinAuthorization(s.enforcer, s.authenticator))
	}

	return r
}
