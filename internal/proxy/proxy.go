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

// FetchMetadata gets metadata using Stale-While-Revalidate caching.
// It checks storage for cacheKey. If found, returns it, and asynchronously fetches upstreamURL,
// compares hashes, and updates storage if necessary.
// If not found, it fetches synchronously, caches, and returns.
func (s *CacheService) FetchMetadata(ctx context.Context, cacheKey string, upstreamURL string, headers map[string]string) ([]byte, error) {
	reader, err := s.storage.Get(ctx, cacheKey)
	if err == nil {
		defer reader.Close()
		content, err := io.ReadAll(reader)
		if err == nil {
			slog.Info("FetchMetadata cache hit", "cacheKey", cacheKey, "component", "proxy")
			
			// Asynchronously revalidate
			go func(bgCtx context.Context, key, url string, hdrs map[string]string) {
				s.revalidateMetadata(bgCtx, key, url, content, hdrs)
			}(context.Background(), cacheKey, upstreamURL, headers)
			
			return content, nil
		}
	}

	slog.Info("FetchMetadata cache miss", "cacheKey", cacheKey, "upstreamURL", upstreamURL, "component", "proxy")

	// Cache miss or error reading cache, do synchronous fetch
	v, err, _ := s.sf.Do(cacheKey, func() (interface{}, error) {
		content, err := s.fetch(ctx, upstreamURL, headers)
		if err != nil {
			return nil, err
		}

		_ = s.storage.Put(ctx, cacheKey, bytes.NewReader(content))
		return content, nil
	})

	if err != nil {
		slog.Error("FetchMetadata failed to fetch", "upstreamURL", upstreamURL, "error", err, "component", "proxy")
		return nil, err
	}

	return v.([]byte), nil
}

func (s *CacheService) revalidateMetadata(ctx context.Context, cacheKey, upstreamURL string, oldContent []byte, headers map[string]string) {
	newContent, err := s.fetch(ctx, upstreamURL, headers)
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

func (s *CacheService) fetch(ctx context.Context, upstreamURL string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
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

func (s *CacheService) GetSingleflightGroup() *singleflight.Group {
	return &s.sf
}
