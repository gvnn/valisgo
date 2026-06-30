package testutil

import (
	"net/http"
	"testing"

	"valisgo/internal/registry/pypi"

	"gorm.io/gorm"
)



// NewPyPITestRouter creates a fully configured PyPI protocol router for testing.
func NewPyPITestRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	packageStore, packageFileStore, st := SetupTestStoresAndStorage(t, db)
	p := pypi.NewPyPIProtocol(packageStore, packageFileStore, st)
	return p.MountRoutes()
}
