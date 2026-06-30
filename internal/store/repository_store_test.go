package store_test

import (
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/store"
	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestRepositoryStore_All(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		registry := &domain.Registry{Name: "test-registry", Format: domain.FormatPyPI}
		tx.Create(registry)

		repo := &domain.Repository{Name: "test-repo", RegistryID: registry.ID}
		tx.Create(repo)

		s := store.NewRepositoryStore(tx)

		repos, _ := s.All()

		found := false
		for _, r := range repos {
			if r.Name == repo.Name {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected to find repo %q in results", repo.Name)
		}
	})
}

func TestRepositoryStore_GetByNameAndRegistryID(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		registry := &domain.Registry{Name: "test-registry-get", Format: domain.FormatPyPI}
		tx.Create(registry)

		repo := &domain.Repository{Name: "test-repo-get", RegistryID: registry.ID}
		tx.Create(repo)

		s := store.NewRepositoryStore(tx)

		foundRepo, _ := s.GetByNameAndRegistryID(repo.Name, registry.ID)

		if foundRepo.ID != repo.ID {
			t.Errorf("expected repo ID %d, got %d", repo.ID, foundRepo.ID)
		}
	})
}
