package server_test

import (
	"testing"

	"valisgo/internal/registry"
	"valisgo/internal/server"

	"github.com/go-chi/chi/v5"
)

type stubProtocol struct{}

func (s *stubProtocol) MountRoutes(repo *registry.Repository) chi.Router {
	return chi.NewRouter()
}

func TestNewServer(t *testing.T) {
	srv := server.NewServer()
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestRegisterProtocol(t *testing.T) {
	srv := server.NewServer()
	srv.RegisterProtocol("pypi", &stubProtocol{})
}

func TestSetupRouterNotNil(t *testing.T) {
	srv := server.NewServer()
	r := srv.SetupRouter()
	if r == nil {
		t.Fatal("SetupRouter returned nil router")
	}
}
