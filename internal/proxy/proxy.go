package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"valisgo/internal/storage"

	"golang.org/x/sync/singleflight"
)

type CacheService struct {
	storage storage.Storage
	client  *http.Client
	sf      singleflight.Group
}

func NewCacheService(storage storage.Storage) *CacheService {
	return &CacheService{
		storage: storage,
		client:  http.DefaultClient,
	}
}

// ServeMetadata serves an endpoint using Stale-While-Revalidate caching.
// It checks storage for cacheKey. If found, serves it. In a goroutine, it fetches upstreamURL,
// applies rewriteFn, compares hashes, and updates storage if necessary.
// If not found, it fetches synchronously, caches, and serves.
func (s *CacheService) ServeMetadata(w http.ResponseWriter, req *http.Request, cacheKey string, upstreamURL string, rewriteFn func([]byte) []byte) {
	reader, err := s.storage.Get(req.Context(), cacheKey)
	if err == nil {
		defer reader.Close()
		content, err := io.ReadAll(reader)
		if err == nil {
			slog.Info("ServeMetadata cache hit", "cacheKey", cacheKey, "component", "proxy")
			// Serve cached version
			_, _ = w.Write(content)

			// Asynchronously revalidate
			go func(ctx context.Context, key, url string) {
				s.revalidateMetadata(ctx, key, url, content, rewriteFn)
			}(context.Background(), cacheKey, upstreamURL)
			return
		}
	}

	slog.Info("ServeMetadata cache miss", "cacheKey", cacheKey, "upstreamURL", upstreamURL, "component", "proxy")

	// Cache miss or error reading cache, do synchronous fetch
	v, err, _ := s.sf.Do(cacheKey, func() (interface{}, error) {
		content, err := s.fetchAndRewrite(req.Context(), upstreamURL, rewriteFn)
		if err != nil {
			return nil, err
		}

		_ = s.storage.Put(req.Context(), cacheKey, bytes.NewReader(content))
		return content, nil
	})

	if err != nil {
		slog.Error("ServeMetadata failed to fetch", "upstreamURL", upstreamURL, "error", err, "component", "proxy")
		http.Error(w, fmt.Sprintf("bad gateway: %v", err), http.StatusBadGateway)
		return
	}

	content := v.([]byte)
	_, _ = w.Write(content)
}

func (s *CacheService) revalidateMetadata(ctx context.Context, cacheKey, upstreamURL string, oldContent []byte, rewriteFn func([]byte) []byte) {
	newContent, err := s.fetchAndRewrite(ctx, upstreamURL, rewriteFn)
	if err != nil {
		slog.Error("Failed to revalidate", "cacheKey", cacheKey, "error", err, "component", "proxy")
		return
	}

	oldHash := sha256.Sum256(oldContent)
	newHash := sha256.Sum256(newContent)

	if oldHash != newHash {
		slog.Info("Content changed, updating cache", "cacheKey", cacheKey, "component", "proxy")
		_ = s.storage.Put(ctx, cacheKey, bytes.NewReader(newContent))
	} else {
		slog.Info("Content unchanged", "cacheKey", cacheKey, "component", "proxy")
	}
}

func (s *CacheService) fetchAndRewrite(ctx context.Context, upstreamURL string, rewriteFn func([]byte) []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if rewriteFn != nil {
		body = rewriteFn(body)
	}

	return body, nil
}

// StreamAndSave downloads the file, writes it to the response, and calls saveFn to save it to DB/Storage.
// The saveFn is responsible for reading the io.Reader until EOF, which simultaneously streams to the client via TeeReader.
func (s *CacheService) StreamAndSave(ctx context.Context, w http.ResponseWriter, upstreamURL string, saveFn func(io.Reader, int64) error) error {
	slog.Info("StreamAndSave fetching", "upstreamURL", upstreamURL, "component", "proxy")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upstream returned %s", resp.Status)
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

	tee := io.TeeReader(resp.Body, w)

	return saveFn(tee, resp.ContentLength)
}
