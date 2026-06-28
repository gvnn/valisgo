package domain

import "gorm.io/gorm"

type RegistryFormat string

const (
	FormatPyPI RegistryFormat = "pypi"
	FormatGo   RegistryFormat = "go"
	FormatFile RegistryFormat = "file"
)

type Registry struct {
	gorm.Model

	Name   string         `gorm:"size:255;uniqueIndex;not null"`
	Format RegistryFormat `gorm:"type:varchar(255);default:'file'"`
}

type RegistryStore interface {
	All() ([]*Registry, error)
}
