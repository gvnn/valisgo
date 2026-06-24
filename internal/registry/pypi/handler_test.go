package pypi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/registry"
	"valisgo/internal/registry/pypi"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	p := &pypi.PyPIProtocol{}
	repo := &registry.Repository{Name: "test-repo", Format: "pypi"}
	return p.MountRoutes(repo)
}

func TestSimplePackageMetadata(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/simple/requests", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "hosted pypi metadata" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestPackageDownload(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/packages/requests/2.31.0/requests-2.31.0-py3-none-any.whl", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "hosted pypi wheel" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestUpload(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
