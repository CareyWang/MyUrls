package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestResolveCache(capacity int, ttl time.Duration) (*resolveCache, *testClock) {
	cache := newResolveCache(capacity, ttl)
	clock := &testClock{current: time.Unix(1_800_000_000, 0)}
	cache.now = clock.now
	return cache, clock
}

func TestResolveCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache, _ := newTestResolveCache(2, time.Hour)
	cache.set("a", "https://example.com/a", permanentLifetime)
	cache.set("b", "https://example.com/b", permanentLifetime)
	_, ok := cache.get("a")
	require.True(t, ok)

	cache.set("c", "https://example.com/c", permanentLifetime)

	_, ok = cache.get("b")
	require.False(t, ok, "b was used least recently")
	longURL, ok := cache.get("a")
	require.True(t, ok)
	require.Equal(t, "https://example.com/a", longURL)
	_, ok = cache.get("c")
	require.True(t, ok)
}

func TestResolveCacheEntryLifetime(t *testing.T) {
	for _, tc := range []struct {
		name         string
		life         lifetime
		wantLifetime time.Duration
	}{
		{"limited by cache ttl", remainingLifetime(365 * 24 * time.Hour), 5 * time.Minute},
		{"limited by the mapping", remainingLifetime(time.Minute), time.Minute},
		{"mapping that never expires", permanentLifetime, 5 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache, clock := newTestResolveCache(2, 5*time.Minute)
			cache.set("key", "https://example.com", tc.life)

			clock.advance(tc.wantLifetime - time.Second)
			_, ok := cache.get("key")
			require.True(t, ok)
			clock.advance(time.Second)
			_, ok = cache.get("key")
			require.False(t, ok)
			require.Empty(t, cache.entries, "an expired entry must not keep its slot")
		})
	}
}

func TestResolveCacheSetReplacesEntry(t *testing.T) {
	cache, _ := newTestResolveCache(1, time.Hour)
	cache.set("key", "https://example.com/old", permanentLifetime)
	cache.set("key", "https://example.com/new", permanentLifetime)

	longURL, ok := cache.get("key")
	require.True(t, ok)
	require.Equal(t, "https://example.com/new", longURL)
	require.Equal(t, 1, cache.order.Len())
}

func TestResolveCacheConcurrentUse(t *testing.T) {
	cache := newResolveCache(8, time.Hour)
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			shortKey := fmt.Sprintf("key%d", i%16)
			cache.set(shortKey, "https://example.com", remainingLifetime(time.Minute))
			cache.get(shortKey)
		}()
	}
	wg.Wait()
	require.LessOrEqual(t, cache.order.Len(), 8)
}

// countingStore counts reads and claims.
type countingStore struct {
	mappingStore
	resolves int
	claims   int
}

func (s *countingStore) Claim(ctx context.Context, shortKey, longURL string, life lifetime) (bool, error) {
	s.claims++
	return s.mappingStore.Claim(ctx, shortKey, longURL, life)
}

func (s *countingStore) Resolve(ctx context.Context, shortKey string) (string, lifetime, error) {
	s.resolves++
	return s.mappingStore.Resolve(ctx, shortKey)
}

func TestMappingsCacheResolvedURLs(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		ctx := context.Background()
		counting := &countingStore{mappingStore: store}
		cache := newResolveCache(8, time.Hour)
		mappings := newShortURLMappings(counting, cache)
		_, err := store.Claim(ctx, "key", "https://example.com", remainingLifetime(time.Hour))
		require.NoError(t, err)

		for range 3 {
			longURL, err := mappings.Resolve(ctx, "key")
			require.NoError(t, err)
			require.Equal(t, "https://example.com", longURL)
		}
		require.Equal(t, 1, counting.resolves)
	})
}

func TestMappingsDoNotCacheMisses(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		ctx := context.Background()
		mappings := newShortURLMappings(store, newResolveCache(8, time.Hour))

		_, err := mappings.Resolve(ctx, "key")
		require.ErrorIs(t, err, errShortURLNotFound)

		_, err = mappings.Create(ctx, "https://example.com", "key")
		require.NoError(t, err)
		longURL, err := mappings.Resolve(ctx, "key")
		require.NoError(t, err)
		require.Equal(t, "https://example.com", longURL)
	})
}

func TestMappingsCacheDoesNotOutliveMapping(t *testing.T) {
	store, clock := newTestSQLiteStore(t)
	ctx := context.Background()
	cache := newResolveCache(8, 365*24*time.Hour)
	cache.now = clock.now
	mappings := newShortURLMappings(store, cache)
	_, err := store.Claim(ctx, "key", "https://example.com/original", remainingLifetime(time.Hour))
	require.NoError(t, err)
	_, err = mappings.Resolve(ctx, "key")
	require.NoError(t, err)

	clock.advance(time.Hour)

	_, err = mappings.Resolve(ctx, "key")
	require.ErrorIs(t, err, errShortURLNotFound)
	_, err = mappings.Create(ctx, "https://example.com/replacement", "key")
	require.NoError(t, err)
	longURL, err := mappings.Resolve(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/replacement", longURL)
}

func TestMappingsWithoutCacheAlwaysReadStore(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	ctx := context.Background()
	counting := &countingStore{mappingStore: store}
	mappings := newShortURLMappings(counting, nil)
	_, err := store.Claim(ctx, "key", "https://example.com", remainingLifetime(time.Hour))
	require.NoError(t, err)

	for range 2 {
		_, err = mappings.Resolve(ctx, "key")
		require.NoError(t, err)
	}
	require.Equal(t, 2, counting.resolves)
}

func TestNewMappingCacheDefaultsToDisabled(t *testing.T) {
	cache, err := newMappingCache()
	require.NoError(t, err)
	require.Nil(t, cache)
}

func TestNewMappingCacheRejectsInvalidSettings(t *testing.T) {
	previousEnabled, previousSize, previousTTL := cacheEnabled, cacheSize, cacheTTL
	t.Cleanup(func() { cacheEnabled, cacheSize, cacheTTL = previousEnabled, previousSize, previousTTL })

	cacheEnabled = true
	cacheSize, cacheTTL = 0, 300
	_, err := newMappingCache()
	require.Error(t, err)
	cacheSize, cacheTTL = 128, -1
	_, err = newMappingCache()
	require.Error(t, err)
	cacheSize, cacheTTL = 16, 60
	cache, err := newMappingCache()
	require.NoError(t, err)
	require.Equal(t, 16, cache.capacity)
	require.Equal(t, time.Minute, cache.ttl)
}

func TestParseEnvironsReadsStorageSettings(t *testing.T) {
	previousType, previousFile := storageType, sqliteFile
	previousEnabled, previousSize, previousTTL := cacheEnabled, cacheSize, cacheTTL
	t.Cleanup(func() {
		storageType, sqliteFile = previousType, previousFile
		cacheEnabled, cacheSize, cacheTTL = previousEnabled, previousSize, previousTTL
	})

	t.Setenv("MYURLS_STORAGE_TYPE", "sqlite")
	t.Setenv("MYURLS_STORAGE_SQLITE_FILE", "/tmp/custom.db")
	t.Setenv("MYURLS_STORAGE_CACHE_ENABLED", "true")
	t.Setenv("MYURLS_STORAGE_CACHE_SIZE", "16")
	t.Setenv("MYURLS_STORAGE_CACHE_TTL", "60")

	require.NoError(t, parseEnvirons())
	require.Equal(t, storageTypeSQLite, storageType)
	require.Equal(t, "/tmp/custom.db", sqliteFile)
	require.True(t, cacheEnabled)
	require.Equal(t, 16, cacheSize)
	require.Equal(t, 60, cacheTTL)
}

func TestParseEnvironsRejectsInvalidCacheSettings(t *testing.T) {
	previousEnabled, previousSize, previousTTL := cacheEnabled, cacheSize, cacheTTL
	t.Cleanup(func() { cacheEnabled, cacheSize, cacheTTL = previousEnabled, previousSize, previousTTL })

	for _, name := range []string{"MYURLS_STORAGE_CACHE_ENABLED", "MYURLS_STORAGE_CACHE_SIZE", "MYURLS_STORAGE_CACHE_TTL"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, "not-a-value")
			require.ErrorContains(t, parseEnvirons(), name)
		})
	}
}
