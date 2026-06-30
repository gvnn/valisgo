package testutil

import (
	"net/http"
	"testing"

	"valisgo/internal/registry/file"

	"gorm.io/gorm"
)



// NewFileTestRouter creates a fully configured File protocol router for testing.
func NewFileTestRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	packageStore, packageFileStore, st := SetupTestStoresAndStorage(t, db)
	p := file.NewFileProtocol(packageStore, packageFileStore, st)
	return p.MountRoutes()
}
