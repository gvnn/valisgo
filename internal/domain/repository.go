package domain

import (
	"context"

	"gorm.io/gorm"
)

const RepoCtxKey = contextKey("repository")

func RepositoryFromContext(ctx context.Context) *Repository {
	repo, ok := ctx.Value(RepoCtxKey).(*Repository)
	if !ok {
		return nil
	}
	return repo
}

type Repository struct {
	gorm.Model

	Name       string `gorm:"size:255;not null;uniqueIndex:idx_name_registry"`
	RegistryID uint   `gorm:"not null;uniqueIndex:idx_name_registry"`
	Registry   Registry
}

type RepositoryStore interface {
	All() ([]*Repository, error)
	GetByNameAndRegistryID(name string, registryID uint) (*Repository, error)
}
