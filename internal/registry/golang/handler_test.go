package golang_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/registry/golang"
	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestPathParsingAndListRouting(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		pkgStore, pkgFileStore, memStorage := testutil.SetupTestStoresAndStorage(t, tx)
		cacheService := proxy.NewCacheService(memStorage)

		proto := golang.NewGoProtocol(pkgStore, pkgFileStore, memStorage, cacheService)
		router := proto.MountRoutes()

		// Create a mock upstream server
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/github.com/gin-gonic/gin/@v/list" {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("v1.2.3\nv1.2.4"))
				return
			}
			http.NotFound(w, r)
		}))
		defer upstream.Close()

		reg, repo := testutil.SetupTestRegistry(tx, "test-golang", domain.FormatGo, "test-repo")
		repo.Type = domain.RepositoryTypeProxy
		repo.UpstreamURL = upstream.URL

		req := httptest.NewRequest(http.MethodGet, "/github.com/gin-gonic/gin/@v/list", nil)
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		if rec.Body.String() != "v1.2.3\nv1.2.4" {
			t.Fatalf("expected list output, got %q", rec.Body.String())
		}
	})
}
