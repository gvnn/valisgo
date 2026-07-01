package pypi_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestSimplePackageMetadata(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewPyPITestRouter(t, tx)

		req := httptest.NewRequest(http.MethodGet, "/simple/", nil)
		reg, repo := testutil.SetupTestRegistry(tx, "test-registry", domain.FormatPyPI, "test-repo")
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "test-repo index") {
			t.Errorf("unexpected body: %q", rec.Body.String())
		}
	})
}

func TestUploadAndDownload(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewPyPITestRouter(t, tx)

		// Upload
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField(":action", "file_upload")
		w.WriteField("name", "requests")
		w.WriteField("version", "2.31.0")

		fw, _ := w.CreateFormFile("content", "requests-2.31.0-py3-none-any.whl")
		fw.Write([]byte("dummy content"))
		w.Close()

		reg, repo := testutil.SetupTestRegistry(tx, "test-registry", domain.FormatPyPI, "test-repo")

		req := httptest.NewRequest(http.MethodPost, "/", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Download
		req2 := httptest.NewRequest(http.MethodGet, "/packages/requests-2.31.0-py3-none-any.whl", nil)
		req2 = testutil.WithRegistryContext(req2, reg, repo)
		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec2.Code)
		}
		if rec2.Body.String() != "dummy content" {
			t.Errorf("unexpected body: %q", rec2.Body.String())
		}
	})
}

func TestUnknownRouteReturns404(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewPyPITestRouter(t, tx)
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})
}

func TestSimpleIndexJSON(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewPyPITestRouter(t, tx)

		req := httptest.NewRequest(http.MethodGet, "/simple/", nil)
		req.Header.Set("Accept", "application/vnd.pypi.simple.v1+json")
		
		reg, repo := testutil.SetupTestRegistry(tx, "test-registry", domain.FormatPyPI, "test-repo")
		
		// Let's create a package so it shows up in the index
		pkg := &domain.Package{Name: "json-test", NormalizedName: "json-test", RepositoryID: repo.ID}
		tx.Create(pkg)

		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		
		body := rec.Body.String()
		if !strings.Contains(body, `"api-version":"1.1"`) {
			t.Errorf("missing api-version 1.1 in json response: %s", body)
		}
		if !strings.Contains(body, `"name":"json-test"`) {
			t.Errorf("missing project json-test in json response: %s", body)
		}
	})
}

func TestSimplePackageJSON(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewPyPITestRouter(t, tx)

		// Upload a file first to have a package file in db
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField(":action", "file_upload")
		w.WriteField("name", "jsontest")
		w.WriteField("version", "1.0.0")

		fw, _ := w.CreateFormFile("content", "jsontest-1.0.0-py3-none-any.whl")
		fw.Write([]byte("dummy content"))
		w.Close()

		reg, repo := testutil.SetupTestRegistry(tx, "test-registry-2", domain.FormatPyPI, "test-repo-2")

		uploadReq := httptest.NewRequest(http.MethodPost, "/", &b)
		uploadReq.Header.Set("Content-Type", w.FormDataContentType())
		uploadReq = testutil.WithRegistryContext(uploadReq, reg, repo)

		uploadRec := httptest.NewRecorder()
		r.ServeHTTP(uploadRec, uploadReq)
		if uploadRec.Code != http.StatusOK {
			t.Fatalf("failed to upload: %d", uploadRec.Code)
		}

		// Now query JSON API
		req := httptest.NewRequest(http.MethodGet, "/simple/jsontest/", nil)
		req.Header.Set("Accept", "application/vnd.pypi.simple.latest+json")
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		
		body := rec.Body.String()
		if !strings.Contains(body, `"api-version":"1.1"`) {
			t.Errorf("missing api-version 1.1 in json response: %s", body)
		}
		if !strings.Contains(body, `"name":"jsontest"`) {
			t.Errorf("missing package name in json response: %s", body)
		}
		if !strings.Contains(body, `"versions":["1.0.0"]`) {
			t.Errorf("missing version 1.0.0 in json response: %s", body)
		}
		// size should be 13 for "dummy content"
		if !strings.Contains(body, `"size":13`) {
			t.Errorf("missing correct size in json response: %s", body)
		}
		if !strings.Contains(body, `"upload-time":`) {
			t.Errorf("missing upload-time in json response: %s", body)
		}
	})
}
