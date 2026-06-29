package pypi_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestSimplePackageMetadata(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewPyPITestRouter(t, tx)

		req := httptest.NewRequest(http.MethodGet, "/simple/", nil)
		reg, repo := testutil.SetupPyPITestDB(tx)
		req = testutil.WithPyPIContext(req, reg, repo)

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

		reg, repo := testutil.SetupPyPITestDB(tx)

		req := httptest.NewRequest(http.MethodPost, "/", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = testutil.WithPyPIContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Download
		req2 := httptest.NewRequest(http.MethodGet, "/packages/requests-2.31.0-py3-none-any.whl", nil)
		req2 = testutil.WithPyPIContext(req2, reg, repo)
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
