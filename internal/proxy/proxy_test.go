package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type mockStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{data: make(map[string][]byte)}
}

func (m *mockStorage) Put(ctx context.Context, key string, data io.Reader) error {
	b, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = b
	return nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func (m *mockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if b, ok := m.data[key]; ok {
		return nopCloser{bytes.NewReader(b)}, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func TestFetchMetadata(t *testing.T) {
	st := newMockStorage()
	svc := NewCacheService(st)

	upstreamCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		if r.Header.Get("Accept") == "application/json" {
			w.Write([]byte(`{"test":"upstream-content"}`))
		} else {
			w.Write([]byte("upstream-content"))
		}
	}))
	defer ts.Close()

	// First call: cache miss, fetches synchronously
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	
	content, err := svc.FetchMetadata(req.Context(), "test-key", ts.URL, map[string]string{"Accept": "application/json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(content) != `{"test":"upstream-content"}` {
		t.Fatalf("expected '{\"test\":\"upstream-content\"}', got '%s'", string(content))
	}
	if upstreamCalls != 1 {
		t.Fatalf("expected 1 upstream call, got %d", upstreamCalls)
	}
	
	// Ensure it was saved in storage
	if b, ok := st.data["test-key"]; !ok || string(b) != `{"test":"upstream-content"}` {
		t.Fatalf("not correctly saved in storage: %v", st.data)
	}

	// Change upstream response to test background revalidation
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.Write([]byte(`{"test":"new-upstream-content"}`))
	}))
	defer ts2.Close()

	// Second call: cache hit, serves from cache, revalidates asynchronously
	content2, err := svc.FetchMetadata(req.Context(), "test-key", ts2.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(content2) != `{"test":"upstream-content"}` { // Should still return old cached content
		t.Fatalf("expected cached '{\"test\":\"upstream-content\"}', got '%s'", string(content2))
	}

	// Give the goroutine time to run
	time.Sleep(100 * time.Millisecond)

	if upstreamCalls != 2 {
		t.Fatalf("expected 2 upstream calls (1 sync, 1 async), got %d", upstreamCalls)
	}

	// The background revalidation should have updated the cache
	if string(st.data["test-key"]) != `{"test":"new-upstream-content"}` {
		t.Fatalf("cache was not updated by revalidation: %s", string(st.data["test-key"]))
	}
}

func TestStreamAndSave(t *testing.T) {
	st := newMockStorage()
	svc := NewCacheService(st)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("artifact-bytes"))
	}))
	defer ts.Close()

	w := httptest.NewRecorder()
	
	var savedBytes []byte
	saveFn := func(r io.Reader, contentLength int64) error {
		b, err := io.ReadAll(r)
		savedBytes = b
		return err
	}

	err := svc.StreamAndSave(context.Background(), w, ts.URL, saveFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Body.String() != "artifact-bytes" {
		t.Fatalf("expected 'artifact-bytes', got '%s'", w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("expected content type header")
	}
	if string(savedBytes) != "artifact-bytes" {
		t.Fatalf("expected 'artifact-bytes' saved, got '%s'", string(savedBytes))
	}
}
