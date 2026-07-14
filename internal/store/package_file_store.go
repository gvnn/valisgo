package store

import (
	"errors"

	"valisgo/internal/domain"

	"gorm.io/gorm"
)

type packageFileStore struct {
	db *gorm.DB
}

func NewPackageFileStore(db *gorm.DB) domain.PackageFileStore {
	return &packageFileStore{db: db}
}

func (s *packageFileStore) Create(file *domain.PackageFile) error {
	return s.db.Create(file).Error
}

func (s *packageFileStore) GetByFilenameAndPackage(filename string, packageID uint) (*domain.PackageFile, error) {
	var file domain.PackageFile
	result := s.db.Where("filename = ? AND package_id = ?", filename, packageID).First(&file)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &file, nil
}

func (s *packageFileStore) GetByFilenameAndRepository(filename string, repositoryID uint) (*domain.PackageFile, error) {
	var file domain.PackageFile
	result := s.db.Joins("Package").Where("\"Package\".repository_id = ? AND filename = ?", repositoryID, filename).First(&file)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &file, nil
}

func (s *packageFileStore) ListByPackage(packageID uint) ([]*domain.PackageFile, error) {
	var files []*domain.PackageFile
	result := s.db.Where("package_id = ?", packageID).Find(&files)
	if result.Error != nil {
		return nil, result.Error
	}
	return files, nil
}
