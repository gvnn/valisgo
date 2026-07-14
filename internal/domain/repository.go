package domain

import (
	"context"
	"time"
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
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name           string `gorm:"size:255;not null;uniqueIndex:idx_name_registry"`
	RegistryID     uint   `gorm:"not null;uniqueIndex:idx_name_registry"`
	Registry       Registry
	Type           RepositoryType      `gorm:"type:varchar(50);not null;default:'local'"`
	UpstreamURL    string              `gorm:"size:255"`
	VirtualMembers []VirtualRepoMember `gorm:"foreignKey:VirtualRepoID"`
}

type VirtualRepoMember struct {
	VirtualRepoID uint       `gorm:"primaryKey"`
	MemberRepoID  uint       `gorm:"primaryKey"`
	Priority      int        `gorm:"not null;default:0"`
	MemberRepo    Repository `gorm:"foreignKey:MemberRepoID"`
}

type RepositoryStore interface {
	All() ([]*Repository, error)
	GetByNameAndRegistryID(name string, registryID uint) (*Repository, error)
	ListByRegistryID(registryID uint) ([]*Repository, error)
	Create(repository *Repository) error
}
