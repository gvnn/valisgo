package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"valisgo/internal/storage"
)

type CacheService struct {
	storage storage.Storage
	client  *http.Client
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
			log.Printf("[PROXY] ServeMetadata cache hit for %s", cacheKey)
			// Serve cached version
			_, _ = w.Write(content)

			// Asynchronously revalidate
			go func(ctx context.Context, key, url string) {
				s.revalidateMetadata(ctx, key, url, content, rewriteFn)
			}(context.Background(), cacheKey, upstreamURL)
			return
		}
	}

	log.Printf("[PROXY] ServeMetadata cache miss for %s, fetching from %s", cacheKey, upstreamURL)

	// Cache miss or error reading cache, do synchronous fetch
	content, err := s.fetchAndRewrite(req.Context(), upstreamURL, rewriteFn)
	if err != nil {
		log.Printf("[PROXY] ServeMetadata failed to fetch %s: %v", upstreamURL, err)
		http.Error(w, fmt.Sprintf("bad gateway: %v", err), http.StatusBadGateway)
		return
	}

	_ = s.storage.Put(req.Context(), cacheKey, bytes.NewReader(content))
	_, _ = w.Write(content)
}

func (s *CacheService) revalidateMetadata(ctx context.Context, cacheKey, upstreamURL string, oldContent []byte, rewriteFn func([]byte) []byte) {
	newContent, err := s.fetchAndRewrite(ctx, upstreamURL, rewriteFn)
	if err != nil {
		log.Printf("[PROXY] Failed to revalidate %s: %v", cacheKey, err)
		return
	}

	oldHash := sha256.Sum256(oldContent)
	newHash := sha256.Sum256(newContent)

	if oldHash != newHash {
		log.Printf("[PROXY] Content for %s changed, updating cache", cacheKey)
		_ = s.storage.Put(ctx, cacheKey, bytes.NewReader(newContent))
	} else {
		log.Printf("[PROXY] Content for %s unchanged", cacheKey)
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
	log.Printf("[PROXY] StreamAndSave fetching %s", upstreamURL)
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
