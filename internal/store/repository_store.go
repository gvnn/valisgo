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
	err := s.db.Preload("Registry").Find(&repositories).Error
	return repositories, err
}

func (s *repositoryStore) GetByName(name string) (*domain.Repository, error) {
	var repo domain.Repository
	err := s.db.Preload("Registry").Where("name = ?", name).First(&repo).Error
	return &repo, err
}
