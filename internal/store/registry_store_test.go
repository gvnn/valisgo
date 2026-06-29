package store_test

import (
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/store"
	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestRegistryStore_All(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		registry := &domain.Registry{Name: "test-registry", Format: domain.FormatPyPI}
		tx.Create(registry)

		s := store.NewRegistryStore(tx)

		registries, _ := s.All()

		if len(registries) != 1 {
			t.Fatalf("expected 1 registry, got %d", len(registries))
		}

		if registries[0].Name != registry.Name {
			t.Errorf("expected registry name 'test-registry', got %q", registries[0].Name)
		}
	})
}
