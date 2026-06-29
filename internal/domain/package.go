package domain

import (
	"gorm.io/gorm"
)

type Package struct {
	gorm.Model

	Name           string `gorm:"size:255;not null"`
	NormalizedName string `gorm:"size:255;not null;index"`
	RepositoryID   uint   `gorm:"not null;uniqueIndex:idx_repository_normalized_name"`
	Repository     Repository

	Files []PackageFile
}

type PackageStore interface {
	GetByNormalizedNameAndRepository(normalizedName string, repositoryID uint) (*Package, error)
	Create(pkg *Package) error
	ListByRepository(repositoryID uint) ([]*Package, error)
}
