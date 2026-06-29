package testutil

import (
	"context"
	"net/http"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/registry/pypi"
	"valisgo/internal/storage"
	"valisgo/internal/store"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
	"gorm.io/gorm"
)

// SetupPyPITestDB creates a dummy registry and repository for PyPI tests.
func SetupPyPITestDB(db *gorm.DB) (*domain.Registry, *domain.Repository) {
	registry := &domain.Registry{Name: "test-registry", Format: domain.FormatPyPI}
	db.Where("name = ?", registry.Name).FirstOrCreate(registry)

	repo := &domain.Repository{Name: "test-repo", RegistryID: registry.ID}
	db.Where("name = ? AND registry_id = ?", repo.Name, registry.ID).FirstOrCreate(repo)

	return registry, repo
}

// WithPyPIContext adds the given registry and repo to the request context.
func WithPyPIContext(req *http.Request, registry *domain.Registry, repo *domain.Repository) *http.Request {
	ctx := context.WithValue(req.Context(), domain.RegistryCtxKey, registry)
	ctx = context.WithValue(ctx, domain.RepoCtxKey, repo)
	return req.WithContext(ctx)
}

// NewPyPITestRouter creates a fully configured PyPI protocol router for testing.
func NewPyPITestRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	packageStore := store.NewPackageStore(db)
	packageFileStore := store.NewPackageFileStore(db)
	
	bucket, err := blob.OpenBucket(context.Background(), "mem://")
	if err != nil {
		t.Fatal(err)
	}
	
	st := storage.NewBlobStorage(bucket)
	p := pypi.NewPyPIProtocol(packageStore, packageFileStore, st)
	
	return p.MountRoutes()
}
