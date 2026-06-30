package file_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func TestUploadAndDownload(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewFileTestRouter(t, tx)
		reg, repo := testutil.SetupTestRegistry(tx, "test-file-registry", domain.FormatFile, "test-file-repo")

		// Upload
		content := []byte("hello world")
		req := httptest.NewRequest(http.MethodPut, "/my/folder/test.txt", bytes.NewReader(content))
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Download
		req2 := httptest.NewRequest(http.MethodGet, "/my/folder/test.txt", nil)
		req2 = testutil.WithRegistryContext(req2, reg, repo)
		
		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec2.Code)
		}
		if rec2.Body.String() != "hello world" {
			t.Errorf("unexpected body: %q", rec2.Body.String())
		}
	})
}

func TestUploadMultipartPOST(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewFileTestRouter(t, tx)
		reg, repo := testutil.SetupTestRegistry(tx, "test-file-registry", domain.FormatFile, "test-file-repo")

		// Upload
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, _ := w.CreateFormFile("file", "test2.txt")
		fw.Write([]byte("multipart content"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/my/folder/", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Download
		req2 := httptest.NewRequest(http.MethodGet, "/my/folder/test2.txt", nil)
		req2 = testutil.WithRegistryContext(req2, reg, repo)
		
		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec2.Code)
		}
		if rec2.Body.String() != "multipart content" {
			t.Errorf("unexpected body: %q", rec2.Body.String())
		}
	})
}

func TestUploadConflict(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewFileTestRouter(t, tx)
		reg, repo := testutil.SetupTestRegistry(tx, "test-file-registry", domain.FormatFile, "test-file-repo")

		// Upload once
		content := []byte("hello world")
		req := httptest.NewRequest(http.MethodPut, "/my/folder/test.txt", bytes.NewReader(content))
		req = testutil.WithRegistryContext(req, reg, repo)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created, got %d", rec.Code)
		}

		// Upload again (conflict)
		req2 := httptest.NewRequest(http.MethodPut, "/my/folder/test.txt", bytes.NewReader(content))
		req2 = testutil.WithRegistryContext(req2, reg, repo)

		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", rec2.Code)
		}
	})
}

func TestDownloadNotFound(t *testing.T) {
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		r := testutil.NewFileTestRouter(t, tx)
		reg, repo := testutil.SetupTestRegistry(tx, "test-file-registry", domain.FormatFile, "test-file-repo")

		req := httptest.NewRequest(http.MethodGet, "/nonexistent.txt", nil)
		req = testutil.WithRegistryContext(req, reg, repo)
		
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}
