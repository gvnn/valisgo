package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/domain"

	"gorm.io/gorm"
)

func TestListRegistries(t *testing.T) {
	runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
		tx.Create(&domain.Registry{Name: "test-reg", Format: domain.FormatPyPI})

		req := httptest.NewRequest(http.MethodGet, "/registries", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		var registries []domain.Registry
		if err := json.NewDecoder(rr.Body).Decode(&registries); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		found := false
		for _, r := range registries {
			if r.Name == "test-reg" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find 'test-reg' in response")
		}
	})
}

func TestCreateRegistry(t *testing.T) {
	createReq := func(name, format string) *http.Request {
		body, _ := json.Marshal(map[string]string{"Name": name, "Format": format})
		return httptest.NewRequest(http.MethodPost, "/registries", bytes.NewReader(body))
	}

	t.Run("success", func(t *testing.T) {
		runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, createReq("new-reg", "go"))

			if rr.Code != http.StatusCreated {
				t.Errorf("expected status code %d, got %d. Body: %s", http.StatusCreated, rr.Code, rr.Body.String())
			}
		})
	})

	t.Run("conflict", func(t *testing.T) {
		runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
			tx.Create(&domain.Registry{Name: "existing-reg", Format: domain.FormatPyPI})

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, createReq("existing-reg", "go"))

			if rr.Code != http.StatusConflict {
				t.Errorf("expected status code %d, got %d. Body: %s", http.StatusConflict, rr.Code, rr.Body.String())
			}
		})
	})
}

func TestListRepositories(t *testing.T) {
	runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
		registry := domain.Registry{Name: "test-reg", Format: domain.FormatPyPI}
		tx.Create(&registry)
		tx.Create(&domain.Repository{Name: "test-repo", RegistryID: registry.ID, Type: domain.RepositoryTypeLocal})

		req := httptest.NewRequest(http.MethodGet, "/repositories", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		var repositories []domain.Repository
		if err := json.NewDecoder(rr.Body).Decode(&repositories); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		found := false
		for _, r := range repositories {
			if r.Name == "test-repo" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find 'test-repo' in response")
		}
	})
}

func TestCreateRepository(t *testing.T) {
	createReq := func(name, registryName, typ string) *http.Request {
		body, _ := json.Marshal(map[string]string{"Name": name, "RegistryName": registryName, "Type": typ})
		return httptest.NewRequest(http.MethodPost, "/repositories", bytes.NewReader(body))
	}

	t.Run("success", func(t *testing.T) {
		runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
			tx.Create(&domain.Registry{Name: "existing-reg", Format: domain.FormatPyPI})

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, createReq("new-repo", "existing-reg", "local"))

			if rr.Code != http.StatusCreated {
				t.Errorf("expected status code %d, got %d. Body: %s", http.StatusCreated, rr.Code, rr.Body.String())
			}
		})
	})

	t.Run("conflict", func(t *testing.T) {
		runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
			registry := domain.Registry{Name: "existing-reg", Format: domain.FormatPyPI}
			tx.Create(&registry)
			tx.Create(&domain.Repository{Name: "existing-repo", RegistryID: registry.ID, Type: domain.RepositoryTypeLocal})

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, createReq("existing-repo", "existing-reg", "local"))

			if rr.Code != http.StatusConflict {
				t.Errorf("expected status code %d, got %d. Body: %s", http.StatusConflict, rr.Code, rr.Body.String())
			}
		})
	})

	t.Run("registry not found", func(t *testing.T) {
		runAPITest(t, func(t *testing.T, tx *gorm.DB, router http.Handler) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, createReq("new-repo", "non-existent-reg", "local"))

			if rr.Code != http.StatusNotFound {
				t.Errorf("expected status code %d, got %d. Body: %s", http.StatusNotFound, rr.Code, rr.Body.String())
			}
		})
	})
}
