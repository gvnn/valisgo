package domain

import "time"

type Package struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name           string `gorm:"size:255;not null"`
	NormalizedName string `gorm:"size:255;not null;uniqueIndex:idx_repository_normalized_name"`
	RepositoryID   uint   `gorm:"not null;uniqueIndex:idx_repository_normalized_name"`
	Repository     Repository

	Files []PackageFile
}

type PackageStore interface {
	GetByNormalizedNameAndRepository(normalizedName string, repositoryID uint) (*Package, error)
	Create(pkg *Package) error
	ListByRepository(repositoryID uint) ([]*Package, error)
	ListDistinctByVirtualRepository(virtualRepoID uint) ([]*Package, error)
}
