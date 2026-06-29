package management

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/domain"
)

type mockRegistryStore struct{}

func (m *mockRegistryStore) All() ([]*domain.Registry, error) {
	return []*domain.Registry{}, nil
}

func (m *mockRegistryStore) GetByName(string) (*domain.Registry, error) {
	return nil, nil
}

func TestListRegistries(t *testing.T) {

	api := &API{
		registryStore: &mockRegistryStore{},
	}
	router := api.MountRoutes()

	req := httptest.NewRequest(http.MethodGet, "/registries", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}
