package pypi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"valisgo/internal/domain"
	"valisgo/internal/registry/pypi"
)

func withRepo(req *http.Request) *http.Request {
	repo := &domain.Repository{Name: "test-repo"}
	ctx := context.WithValue(req.Context(), domain.RepoCtxKey, repo)
	return req.WithContext(ctx)
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	p := &pypi.PyPIProtocol{}
	return p.MountRoutes()
}

func TestSimplePackageMetadata(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/simple/requests", nil)
	req = withRepo(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "hosted pypi metadata for package 'requests' in repository 'test-repo'" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestPackageDownload(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/packages/requests/2.31.0/requests-2.31.0-py3-none-any.whl", nil)
	req = withRepo(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "Downloading wheel from repository: test-repo" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestUpload(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withRepo(req)
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
