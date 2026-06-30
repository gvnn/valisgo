package testutil

import (
	"context"
	"net/http"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/storage"
	"valisgo/internal/store"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
	"gorm.io/gorm"
)

// SetupTestRegistry creates a dummy registry and repository.
func SetupTestRegistry(db *gorm.DB, name string, format domain.RegistryFormat, repoName string) (*domain.Registry, *domain.Repository) {
	registry := &domain.Registry{Name: name, Format: format}
	db.Where("name = ?", registry.Name).FirstOrCreate(registry)

	repo := &domain.Repository{Name: repoName, RegistryID: registry.ID}
	db.Where("name = ? AND registry_id = ?", repo.Name, registry.ID).FirstOrCreate(repo)

	return registry, repo
}

// WithRegistryContext adds the given registry and repo to the request context.
func WithRegistryContext(req *http.Request, registry *domain.Registry, repo *domain.Repository) *http.Request {
	ctx := context.WithValue(req.Context(), domain.RegistryCtxKey, registry)
	ctx = context.WithValue(ctx, domain.RepoCtxKey, repo)
	return req.WithContext(ctx)
}

// SetupTestStoresAndStorage creates in-memory storage and db stores for testing protocols.
func SetupTestStoresAndStorage(t *testing.T, db *gorm.DB) (domain.PackageStore, domain.PackageFileStore, storage.Storage) {
	t.Helper()
	packageStore := store.NewPackageStore(db)
	packageFileStore := store.NewPackageFileStore(db)
	
	bucket, err := blob.OpenBucket(context.Background(), "mem://")
	if err != nil {
		t.Fatal(err)
	}
	
	st := storage.NewBlobStorage(bucket)
	return packageStore, packageFileStore, st
}
