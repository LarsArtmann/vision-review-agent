package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"
)

// cacheData is the on-disk cache format for the remote catalog.
type cacheData struct {
	ETag      string             `json:"etag"`
	Providers []catwalk.Provider `json:"providers"`
}

// Sync manages remote catalog synchronization with ETag-based conditional
// requests and local file caching. It follows the pattern established by
// Crush's catwalkSync: try remote, fall back to cache, then embedded.
type Sync struct {
	client    *catwalk.Client
	cachePath string
}

// NewSync creates a Sync that uses the catwalk HTTP client and persists
// cache to the given path.
func NewSync(cachePath string) *Sync {
	return &Sync{
		client:    catwalk.New(),
		cachePath: cachePath,
	}
}

// DefaultCachePath returns the XDG-compliant path for the catalog cache:
// - Linux: ~/.config/vision/catwalk-cache.json
// - macOS: ~/Library/Application Support/vision/catwalk-cache.json
// - Windows: %APPDATA%\vision\catwalk-cache.json.
func DefaultCachePath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "catwalk-cache.json"
	}

	return filepath.Join(dir, "vision", "catwalk-cache.json")
}

// Fetch retrieves the catalog from the remote server using ETag-based
// conditional requests. The flow is:
//  1. Read cached ETag from disk
//  2. Send conditional GET to remote (If-None-Match: ETag)
//  3. On 200: write new data to cache, return it
//  4. On 304 (NotModified): return cached data
//  5. On error (network, timeout): try cached data, then fall back to embedded
//
// This method never returns an error — it always returns a valid provider list.
func (s *Sync) Fetch(ctx context.Context) []catwalk.Provider {
	etag := s.readCachedETag()

	providers, err := s.client.GetProviders(ctx, etag)
	if err == nil {
		_ = s.writeCache(providers)

		return providers
	}

	if errors.Is(err, catwalk.ErrNotModified) {
		if cached := s.readCachedProviders(); cached != nil {
			return cached
		}
	}

	if cached := s.readCachedProviders(); cached != nil {
		return cached
	}

	return embedded.GetAll()
}

func (s *Sync) readCachedETag() string {
	data, err := s.readCache()
	if err != nil {
		return ""
	}

	return data.ETag
}

func (s *Sync) readCachedProviders() []catwalk.Provider {
	data, err := s.readCache()
	if err != nil {
		return nil
	}

	return data.Providers
}

func (s *Sync) readCache() (*cacheData, error) {
	raw, err := os.ReadFile(s.cachePath)
	if err != nil {
		return nil, fmt.Errorf("read cache %s: %w", s.cachePath, err)
	}

	var data cacheData
	if err := json.Unmarshal(raw, &data); err != nil {
		_ = os.Remove(s.cachePath)

		return nil, fmt.Errorf("corrupted cache, removed: %w", err)
	}

	return &data, nil
}

func (s *Sync) writeCache(providers []catwalk.Provider) error {
	raw, err := json.Marshal(providers)
	if err != nil {
		return fmt.Errorf("marshal providers: %w", err)
	}

	etag := catwalk.Etag(raw)

	data := cacheData{
		ETag:      etag,
		Providers: providers,
	}

	fullRaw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	dir := filepath.Dir(s.cachePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create cache dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "catwalk-cache-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(fullRaw); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), s.cachePath); err != nil {
		return fmt.Errorf("rename temp to cache: %w", err)
	}

	return nil
}
