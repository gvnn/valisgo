package domain

import "time"

type PackageFile struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	PackageID uint `gorm:"not null;uniqueIndex:idx_package_filename"`
	Package   Package

	Version  string `gorm:"size:255;not null"`
	Filename string `gorm:"size:255;not null;uniqueIndex:idx_package_filename"`
	Hash     string `gorm:"size:255;not null"`
	Size     int64  `gorm:"not null"`
	BlobKey  string `gorm:"size:255;not null"`
}

type PackageFileStore interface {
	Create(file *PackageFile) error
	GetByFilenameAndPackage(filename string, packageID uint) (*PackageFile, error)
	GetByFilenameAndRepository(filename string, repositoryID uint) (*PackageFile, error)
	ListByPackage(packageID uint) ([]*PackageFile, error)
}
