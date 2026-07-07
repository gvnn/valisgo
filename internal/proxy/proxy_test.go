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

func TestServeMetadata(t *testing.T) {
	st := newMockStorage()
	svc := NewCacheService(st)

	upstreamCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		w.Write([]byte("upstream-content"))
	}))
	defer ts.Close()

	// First call: cache miss, fetches synchronously
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	svc.ServeMetadata(w, req, "test-key", ts.URL, nil)

	if w.Body.String() != "upstream-content" {
		t.Fatalf("expected 'upstream-content', got '%s'", w.Body.String())
	}
	if upstreamCalls != 1 {
		t.Fatalf("expected 1 upstream call, got %d", upstreamCalls)
	}
	
	// Ensure it was saved in storage
	if b, ok := st.data["test-key"]; !ok || string(b) != "upstream-content" {
		t.Fatalf("not correctly saved in storage: %v", st.data)
	}

	// Second call: cache hit, serves from cache, revalidates asynchronously
	w2 := httptest.NewRecorder()
	svc.ServeMetadata(w2, req, "test-key", ts.URL, func(b []byte) []byte {
		return append(b, []byte("-rewritten")...)
	})

	if w2.Body.String() != "upstream-content" {
		t.Fatalf("expected cached 'upstream-content', got '%s'", w2.Body.String())
	}

	// Give the goroutine time to run
	time.Sleep(100 * time.Millisecond)

	if upstreamCalls != 2 {
		t.Fatalf("expected 2 upstream calls (1 sync, 1 async), got %d", upstreamCalls)
	}

	// The background revalidation should have updated the cache because of rewriteFn
	if string(st.data["test-key"]) != "upstream-content-rewritten" {
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
