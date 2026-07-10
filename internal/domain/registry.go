package domain

import (
	"context"

	"gorm.io/gorm"
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
	gorm.Model

	Name   string         `gorm:"size:255;uniqueIndex;not null"`
	Format RegistryFormat `gorm:"type:varchar(255);default:'file'"`
}

type RegistryStore interface {
	All() ([]*Registry, error)
	GetByName(name string) (*Registry, error)
}
