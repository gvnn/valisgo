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

		found := false
		for _, r := range registries {
			if r.Name == registry.Name {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected to find registry %q in results", registry.Name)
		}
	})
}
