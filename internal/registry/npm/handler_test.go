package npm_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestProxyMetadata(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewNPMTestRouter(t, tx)

		// Create a mock upstream server
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/is-odd" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"name": "is-odd",
					"versions": {
						"0.1.0": {
							"name": "is-odd",
							"version": "0.1.0",
							"dist": {
								"tarball": "https://registry.npmjs.org/is-odd/-/is-odd-0.1.0.tgz"
							}
						}
					}
				}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer upstream.Close()

		req := httptest.NewRequest(http.MethodGet, "/is-odd", nil)
		reg, repo := testutil.SetupTestRegistry(tx, "test-registry", domain.FormatNPM, "test-repo")
		repo.Type = domain.RepositoryTypeProxy
		repo.UpstreamURL = upstream.URL
		
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var meta map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		versions := meta["versions"].(map[string]interface{})
		v010 := versions["0.1.0"].(map[string]interface{})
		dist := v010["dist"].(map[string]interface{})
		tarball := dist["tarball"].(string)

		expectedTarball := "http://example.com/registries/test-registry/repositories/test-repo/is-odd/-/is-odd-0.1.0.tgz?pkg=is-odd&upstream=https%3A%2F%2Fregistry.npmjs.org%2Fis-odd%2F-%2Fis-odd-0.1.0.tgz"
		if tarball != expectedTarball {
			t.Errorf("expected tarball to be %q, got %q", expectedTarball, tarball)
		}
	})
}
