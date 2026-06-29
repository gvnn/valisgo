package store

import (
	"valisgo/internal/domain"

	"gorm.io/gorm"
)

type registryStore struct {
	*BaseStore[domain.Registry]
}

func NewRegistryStore(db *gorm.DB) domain.RegistryStore {
	return &registryStore{
		BaseStore: &BaseStore[domain.Registry]{db: db},
	}
}

func (s *registryStore) All() ([]*domain.Registry, error) {
	var registries []*domain.Registry
	err := s.db.Find(&registries).Error
	return registries, err
}

func (s *registryStore) GetByName(name string) (*domain.Registry, error) {
	var registry domain.Registry

	err := s.db.Where("name = ?", name).First(&registry).Error
	if err != nil {
		return nil, err
	}

	return &registry, nil
}
