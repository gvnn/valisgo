package server

import (
	"github.com/casbin/casbin/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	enforcer *casbin.Enforcer
}

func NewServer(enforcer *casbin.Enforcer) *Server {
	return &Server{
		enforcer: enforcer,
	}
}

func (s *Server) SetupRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if s.enforcer != nil {
		r.Use(CasbinAuthorization(s.enforcer))
	}

	return r
}
