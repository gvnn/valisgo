package store

import "gorm.io/gorm"

type BaseStore[T any] struct {
	db *gorm.DB
}
