package testutil

import (
	"net/http"
	"testing"

	"valisgo/internal/proxy"
	"valisgo/internal/registry/pypi"

	"gorm.io/gorm"
)



// NewPyPITestRouter creates a fully configured PyPI protocol router for testing.
func NewPyPITestRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	packageStore, packageFileStore, st := SetupTestStoresAndStorage(t, db)
	cacheService := proxy.NewCacheService(st)
	p := pypi.NewPyPIProtocol(packageStore, packageFileStore, st, cacheService)
	return p.MountRoutes()
}
