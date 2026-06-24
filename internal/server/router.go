package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"valisgo/internal/registry"
)

type Server struct {
	protocols    map[string]registry.Protocol
	repositories []registry.Repository
}

func NewServer() *Server {
	return &Server{
		protocols: make(map[string]registry.Protocol),
	}
}

func (s *Server) RegisterProtocol(format string, p registry.Protocol) {
	s.protocols[format] = p
}

func (s *Server) RegisterRepository(repo registry.Repository) {
	s.repositories = append(s.repositories, repo)
}

func (s *Server) SetupRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	for i := range s.repositories {
		repo := s.repositories[i]
		proto, ok := s.protocols[repo.Format]
		if !ok {
			continue
		}
		r.Mount("/"+repo.Name, proto.MountRoutes(&repo))
	}

	return r
}
