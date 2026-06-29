package store

import (
	"valisgo/internal/domain"

	"gorm.io/gorm"
)

type packageStore struct {
	db *gorm.DB
}

func NewPackageStore(db *gorm.DB) domain.PackageStore {
	return &packageStore{db: db}
}

func (s *packageStore) GetByNormalizedNameAndRepository(normalizedName string, repositoryID uint) (*domain.Package, error) {
	var pkg domain.Package
	result := s.db.Where("normalized_name = ? AND repository_id = ?", normalizedName, repositoryID).First(&pkg)
	if result.Error != nil {
		return nil, result.Error
	}
	return &pkg, nil
}

func (s *packageStore) Create(pkg *domain.Package) error {
	return s.db.Create(pkg).Error
}

func (s *packageStore) ListByRepository(repositoryID uint) ([]*domain.Package, error) {
	var pkgs []*domain.Package
	err := s.db.Where("repository_id = ?", repositoryID).Find(&pkgs).Error
	return pkgs, err
}
