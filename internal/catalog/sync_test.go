package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"
	"github.com/stretchr/testify/require"
)

func TestSyncWriteCacheAndReadBack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "sub", "catwalk-cache.json")

	syncer := NewSync(cachePath)

	providers := []catwalk.Provider{
		{Name: "Test", ID: "test", Type: catwalk.TypeOpenAI},
		{Name: "Test2", ID: "test2", Type: catwalk.TypeAnthropic},
	}

	err := syncer.writeCache(providers)
	require.NoError(t, err)

	data, err := syncer.readCache()
	require.NoError(t, err)
	require.Equal(t, "Test", data.Providers[0].Name)
	require.NotEmpty(t, data.ETag, "ETag must be computed and stored")
}

func TestSyncReadCacheMissingFile(t *testing.T) {
	t.Parallel()

	syncer := NewSync("/nonexistent/path/cache.json")

	_, err := syncer.readCache()
	require.Error(t, err)
}

func TestSyncReadCacheCorruptedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	require.NoError(t, os.WriteFile(cachePath, []byte("{not valid json}"), 0o600))

	syncer := NewSync(cachePath)

	_, err := syncer.readCache()
	require.Error(t, err)

	// Corrupted cache must be deleted
	_, statErr := os.Stat(cachePath)
	require.True(t, os.IsNotExist(statErr), "corrupted cache file must be removed")
}

func TestSyncReadCachedETagMissingFile(t *testing.T) {
	t.Parallel()

	syncer := NewSync("/nonexistent/cache.json")
	etag := syncer.readCachedETag()
	require.Empty(t, etag)
}

func TestSyncReadCachedProvidersMissingFile(t *testing.T) {
	t.Parallel()

	syncer := NewSync("/nonexistent/cache.json")
	providers := syncer.readCachedProviders()
	require.Nil(t, providers)
}

func TestSyncWriteCacheCreatesParentDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	deepPath := filepath.Join(dir, "a", "b", "c", "cache.json")

	syncer := NewSync(deepPath)

	err := syncer.writeCache([]catwalk.Provider{{Name: "X", ID: "x"}})
	require.NoError(t, err)

	_, err = os.Stat(deepPath)
	require.NoError(t, err, "cache file must exist at deep path")
}

func TestSyncFetchFallsBackToEmbeddedOnNoServer(t *testing.T) {
	t.Parallel()

	// Point to a non-existent server — Fetch must fall back to embedded
	syncer := NewSync(filepath.Join(t.TempDir(), "cache.json"))

	ctx := context.Background()
	providers := syncer.Fetch(ctx)

	require.NotEmpty(t, providers, "Fetch must always return providers (embedded fallback)")
	require.Greater(t, len(providers), 10, "embedded catalog must have many providers")
}

func TestSyncFetchUsesCachedDataOnNetworkError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	// Write cached data first
	cached := []catwalk.Provider{
		{Name: "Cached Provider", ID: "cached-test", Type: catwalk.TypeOpenAI},
	}
	syncer := NewSync(cachePath)
	require.NoError(t, syncer.writeCache(cached))

	// Fetch with no server available — should use cached data
	providers := syncer.Fetch(context.Background())

	require.Len(t, providers, 1)
	require.Equal(t, "Cached Provider", providers[0].Name)
}

func TestSyncFetchFallsBackToEmbeddedWhenNoCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "nonexistent.json")

	syncer := NewSync(cachePath)

	providers := syncer.Fetch(context.Background())

	embeddedProviders := embedded.GetAll()
	require.Len(t, providers, len(embeddedProviders),
		"Fetch must fall back to embedded when no cache and no server")
}

func TestDefaultCachePathContainsVision(t *testing.T) {
	t.Parallel()

	path := DefaultCachePath()
	require.Contains(t, path, "vision")
	require.Contains(t, path, "catwalk-cache.json")
}

func TestSyncWriteCacheStoresValidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	syncer := NewSync(cachePath)

	providers := []catwalk.Provider{
		{Name: "Test", ID: "test", Type: catwalk.TypeOpenAI},
	}

	require.NoError(t, syncer.writeCache(providers))

	raw, err := os.ReadFile(cachePath)
	require.NoError(t, err)

	var data cacheData
	require.NoError(t, json.Unmarshal(raw, &data))
	require.NotEmpty(t, data.ETag)
	require.Len(t, data.Providers, 1)
	require.Equal(t, "Test", data.Providers[0].Name)
}
