package domain

import "gorm.io/gorm"

type Repository struct {
	gorm.Model

	Name       string `gorm:"size:255;not null"`
	RegistryID uint   `gorm:"not null"`
	Registry   Registry
}

type RepositoryStore interface {
	All() ([]*Repository, error)
}
