package domain

import (
	"context"
	"time"
)

type contextKey string

const RegistryCtxKey = contextKey("registry")

func RegistryFromContext(ctx context.Context) *Registry {
	reg, _ := ctx.Value(RegistryCtxKey).(*Registry)
	return reg
}

type RegistryFormat string

const (
	FormatPyPI RegistryFormat = "pypi"
	FormatGo   RegistryFormat = "go"
	FormatNPM  RegistryFormat = "npm"
	FormatFile RegistryFormat = "file"
)

type Registry struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name   string         `gorm:"size:255;uniqueIndex;not null"`
	Format RegistryFormat `gorm:"type:varchar(255);default:'file'"`
}

type RegistryStore interface {
	All() ([]*Registry, error)
	GetByName(name string) (*Registry, error)
	Create(registry *Registry) error
}
