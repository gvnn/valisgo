package testutil

import (
	"net/http"
	"testing"

	"valisgo/internal/proxy"
	"valisgo/internal/registry/npm"

	"gorm.io/gorm"
)

// NewNPMTestRouter creates a fully configured NPM protocol router for testing.
func NewNPMTestRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	packageStore, packageFileStore, st := SetupTestStoresAndStorage(t, db)
	cacheService := proxy.NewCacheService(st)
	p := npm.NewNPMProtocol(packageStore, packageFileStore, st, cacheService)
	return p.MountRoutes()
}
