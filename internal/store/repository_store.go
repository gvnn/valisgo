package store

import (
	"valisgo/internal/domain"

	"gorm.io/gorm"
)

type repositoryStore struct {
	*BaseStore[domain.Repository]
}

func NewRepositoryStore(db *gorm.DB) domain.RepositoryStore {
	return &repositoryStore{
		BaseStore: &BaseStore[domain.Repository]{db: db},
	}
}

func (s *repositoryStore) All() ([]*domain.Repository, error) {
	var repositories []*domain.Repository
	err := s.db.Preload("Registry").
		Preload("VirtualMembers", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority desc")
		}).
		Preload("VirtualMembers.MemberRepo").
		Find(&repositories).Error
	return repositories, err
}

func (s *repositoryStore) GetByNameAndRegistryID(name string, registryID uint) (*domain.Repository, error) {
	var repo domain.Repository

	err := s.db.Preload("Registry").
		Preload("VirtualMembers", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority desc")
		}).
		Preload("VirtualMembers.MemberRepo").
		Where("name = ? AND registry_id = ?", name, registryID).
		First(&repo).Error

	if err != nil {
		return nil, err
	}

	return &repo, nil
}
