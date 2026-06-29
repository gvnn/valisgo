package server_test

import (
	"testing"

	"valisgo/internal/server"

	"github.com/go-chi/chi/v5"
)

type stubProtocol struct{}

func (s *stubProtocol) MountRoutes() chi.Router {
	return chi.NewRouter()
}

func TestSetupRouterNotNil(t *testing.T) {
	srv := server.NewServer()
	r := srv.SetupRouter()
	if r == nil {
		t.Fatal("SetupRouter returned nil router")
	}
}
