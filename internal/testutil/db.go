package testutil

import (
	"sync"
	"testing"

	"valisgo/internal/database"
	"valisgo/internal/env"

	"gorm.io/gorm"
)

var getTestDB = sync.OnceValues(func() (*gorm.DB, error) {
	driver := env.GetOrDefault("TEST_DB_DRIVER", env.GetOrDefault("DB_DRIVER", "postgres"))
	dsn := env.GetOrDefault("TEST_DB_DSN", env.GetOrDefault("DB_DSN", "postgres://user:pass@localhost:5432/valisgo?sslmode=disable"))

	return database.NewConnection(driver, dsn)
})

func RunInTransaction(t *testing.T, testFn func(tx *gorm.DB)) {
	t.Helper()

	db, err := getTestDB()
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		tx.Rollback()
	}()

	testFn(tx)
}
