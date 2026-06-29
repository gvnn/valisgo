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

		if len(repos) != 1 {
			t.Fatalf("expected 1 repo, got %d", len(repos))
		}

		if repos[0].Name != repo.Name {
			t.Errorf("expected repo name 'test-repo', got %q", repos[0].Name)
		}
	})
}

func TestRepositoryStore_GetByName(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		registry := &domain.Registry{Name: "test-registry-get", Format: domain.FormatPyPI}
		tx.Create(registry)

		repo := &domain.Repository{Name: "test-repo-get", RegistryID: registry.ID}
		tx.Create(repo)

		s := store.NewRepositoryStore(tx)

		foundRepo, _ := s.GetByName(repo.Name)

		if foundRepo.ID != repo.ID {
			t.Errorf("expected repo ID %d, got %d", repo.ID, foundRepo.ID)
		}
	})
}
