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

type RepositoryType string

const (
	RepositoryTypeLocal   RepositoryType = "local"
	RepositoryTypeProxy   RepositoryType = "proxy"
	RepositoryTypeVirtual RepositoryType = "virtual"
)

type Repository struct {
	gorm.Model

	Name           string `gorm:"size:255;not null;uniqueIndex:idx_name_registry"`
	RegistryID     uint   `gorm:"not null;uniqueIndex:idx_name_registry"`
	Registry       Registry
	Type           RepositoryType      `gorm:"type:varchar(50);not null;default:'local'"`
	UpstreamURL    string              `gorm:"size:255"`
	VirtualMembers []VirtualRepoMember `gorm:"foreignKey:VirtualRepoID"`
}

type VirtualRepoMember struct {
	VirtualRepoID uint `gorm:"primaryKey"`
	MemberRepoID  uint `gorm:"primaryKey"`
	Priority      int  `gorm:"not null;default:0"`
}

type RepositoryStore interface {
	All() ([]*Repository, error)
	GetByNameAndRegistryID(name string, registryID uint) (*Repository, error)
}
