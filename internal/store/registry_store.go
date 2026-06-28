package store

import (
	"valisgo/internal/domain"

	"gorm.io/gorm"
)

type registryStore struct {
	db *gorm.DB
}

func NewRegistryStore(db *gorm.DB) domain.RegistryStore {
	return &registryStore{db: db}
}

func (s *registryStore) All() ([]*domain.Registry, error) {
	var registries []*domain.Registry

	err := s.db.Find(&registries).Error
	if err != nil {
		return nil, err
	}

	return registries, nil
}
