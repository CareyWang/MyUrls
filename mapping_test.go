package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var errStorageDown = errors.New("storage unavailable")

func newTestRedisMappingsAt(t testing.TB, addr string) *shortURLMappings {
	t.Helper()
	mappings := newShortURLMappings(newRedisStore(addr, ""), nil)
	t.Cleanup(func() {
		if err := mappings.Close(); err != nil {
			t.Errorf("close short URL mappings: %v", err)
		}
	})
	return mappings
}

// newTestMappings returns mappings over Redis, the production default, for tests
// whose behavior does not depend on the store.
func newTestMappings(t testing.TB) *shortURLMappings {
	t.Helper()
	return newTestRedisMappingsAt(t, miniredis.RunT(t).Addr())
}

// forEachMappings runs a behavior that mappings must show on every storage
// backend. advance moves the backend's clock forward. forEachStore closes the
// store, so the test must not close the mappings.
func forEachMappings(t *testing.T, test func(t *testing.T, mappings *shortURLMappings, advance func(time.Duration))) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		test(t, newShortURLMappings(store, nil), advance)
	})
}

// failingStore fails every call except Close, to simulate a storage outage.
type failingStore struct {
	mappingStore
	err error
}

func (s *failingStore) Claim(context.Context, string, string, lifetime) (bool, error) {
	return false, s.err
}

func (s *failingStore) Resolve(context.Context, string) (string, lifetime, error) {
	return "", lifetime{}, s.err
}

func (s *failingStore) Ping(context.Context) error {
	return s.err
}

// failStorage makes every later call of the mappings to its store fail.
func failStorage(mappings *shortURLMappings) {
	mappings.store = &failingStore{mappingStore: mappings.store, err: errStorageDown}
}

// seedMapping stores a mapping the way another service instance would.
func seedMapping(t testing.TB, store mappingStore, shortKey, longURL string, life lifetime) {
	t.Helper()
	claimed, err := store.Claim(context.Background(), shortKey, longURL, life)
	require.NoError(t, err)
	require.True(t, claimed)
}

func requireMapping(t testing.TB, mappings *shortURLMappings, shortKey, wantLongURL string) {
	t.Helper()
	longURL, err := mappings.Resolve(context.Background(), shortKey)
	require.NoError(t, err)
	require.Equal(t, wantLongURL, longURL)
}

func requireNotFound(t testing.TB, mappings *shortURLMappings, shortKey string) {
	t.Helper()
	_, err := mappings.Resolve(context.Background(), shortKey)
	require.ErrorIs(t, err, errShortURLNotFound)
}

type capturedRedisLogs struct {
	mu   sync.Mutex
	logs strings.Builder
}

func (c *capturedRedisLogs) Printf(_ context.Context, format string, v ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(&c.logs, format+"\n", v...)
}

func TestNewShortURLMappingsInitializesBothClientsCleanly(t *testing.T) {
	// go-redis cannot hand back its current logger, so the capture stays installed.
	captured := &capturedRedisLogs{}
	redis.SetLogger(captured)

	mappings := newTestMappings(t)
	require.NoError(t, mappings.Ping(context.Background()))
	_, err := mappings.Create(context.Background(), "https://example.com", "custom")
	require.NoError(t, err)

	captured.mu.Lock()
	defer captured.mu.Unlock()
	require.NotContains(t, captured.logs.String(), "failed to initialize maintnotifications")
}

func TestPingShortURLMappings(t *testing.T) {
	forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
		require.NoError(t, mappings.Ping(context.Background()))
	})
}

func TestPingShortURLMappingsReportsStorageFailure(t *testing.T) {
	mappings := newTestMappings(t)
	failStorage(mappings)
	require.ErrorIs(t, mappings.Ping(context.Background()), errStorageDown)
}

func TestResolveShortURL(t *testing.T) {
	forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
		ctx := context.Background()
		key, err := mappings.Create(ctx, "https://example.com", "testKey")
		require.NoError(t, err)
		requireMapping(t, mappings, key, "https://example.com")

		longURL, err := mappings.Resolve(ctx, "missing")
		require.ErrorIs(t, err, errShortURLNotFound)
		require.Empty(t, longURL)
	})
}

func TestResolveShortURLReportsStorageFailure(t *testing.T) {
	mappings := newTestMappings(t)
	ctx := context.Background()
	key, err := mappings.Create(ctx, "https://example.com", "testKey")
	require.NoError(t, err)

	failStorage(mappings)
	longURL, err := mappings.Resolve(ctx, key)
	require.ErrorIs(t, err, errStorageDown)
	require.NotErrorIs(t, err, errShortURLNotFound)
	require.Empty(t, longURL)
}

func TestCreateShortURLExpiration(t *testing.T) {
	for _, requestedKey := range []string{"custom", ""} {
		t.Run(fmt.Sprintf("key=%q", requestedKey), func(t *testing.T) {
			forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, advance func(time.Duration)) {
				ctx := context.Background()
				key, err := mappings.Create(ctx, "https://example.com/original", requestedKey)
				require.NoError(t, err)
				if requestedKey == "" {
					require.Regexp(t, `^[0-9a-zA-Z]{7}$`, key)
				} else {
					require.Equal(t, requestedKey, key)
				}
				requireMapping(t, mappings, key, "https://example.com/original")

				advance(time.Hour)
				_, err = mappings.Create(ctx, "https://example.com/replacement", key)
				require.ErrorIs(t, err, errShortKeyExists)
				requireMapping(t, mappings, key, "https://example.com/original")

				advance(365*24*time.Hour - time.Hour - time.Second)
				requireMapping(t, mappings, key, "https://example.com/original")

				advance(time.Second)
				requireNotFound(t, mappings, key)
				_, err = mappings.Create(ctx, "https://example.com/replacement", key)
				require.NoError(t, err)
				requireMapping(t, mappings, key, "https://example.com/replacement")
			})
		})
	}
}

func TestCreateShortURLCollisionRetries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		collisions int
		wantCalls  int
		wantErr    error
	}{
		{name: "first retry succeeds", collisions: 1, wantCalls: 2},
		{name: "third retry succeeds", collisions: 3, wantCalls: 4},
		{name: "three retries exhausted", collisions: 4, wantCalls: 4, wantErr: errShortKeyRetriesExhausted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
				ctx := context.Background()
				seedMapping(t, mappings.store, "taken", "https://example.com/original", remainingLifetime(time.Hour))
				calls := 0
				mappings.generateKey = func() (string, error) {
					calls++
					if calls <= tc.collisions {
						return "taken", nil
					}
					return "fresh", nil
				}

				key, err := mappings.Create(ctx, "https://example.com/new", "")
				require.ErrorIs(t, err, tc.wantErr)
				require.Equal(t, tc.wantCalls, calls)
				requireMapping(t, mappings, "taken", "https://example.com/original")
				_, life, err := mappings.store.Resolve(ctx, "taken")
				require.NoError(t, err)
				require.Equal(t, remainingLifetime(time.Hour), life, "a collision must not change the holder's lifetime")
				if tc.wantErr != nil {
					require.Empty(t, key)
				} else {
					require.Equal(t, "fresh", key)
					requireMapping(t, mappings, key, "https://example.com/new")
				}
			})
		})
	}
}

func TestCreateShortURLCustomCollisionDoesNotGenerate(t *testing.T) {
	forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
		ctx := context.Background()
		seedMapping(t, mappings.store, "custom", "original", permanentLifetime)
		mappings.generateKey = func() (string, error) {
			t.Fatal("a custom key must not trigger automatic generation")
			return "", nil
		}
		key, err := mappings.Create(ctx, "replacement", "custom")
		require.ErrorIs(t, err, errShortKeyExists)
		require.Empty(t, key)
		requireMapping(t, mappings, "custom", "original")
	})
}

func TestCreateShortURLConcurrentClaims(t *testing.T) {
	forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
		ctx := context.Background()
		const requests = 32
		type result struct {
			url string
			key string
			err error
		}
		results := make(chan result, requests)
		start := make(chan struct{})
		for i := 0; i < requests; i++ {
			go func(i int) {
				<-start
				url := fmt.Sprintf("https://example.com/%d", i)
				key, err := mappings.Create(ctx, url, "contended")
				results <- result{url: url, key: key, err: err}
			}(i)
		}
		close(start)

		winners := 0
		var winningURL string
		for i := 0; i < requests; i++ {
			r := <-results
			if r.err == nil {
				winners++
				winningURL = r.url
				require.Equal(t, "contended", r.key)
			} else {
				require.ErrorIs(t, r.err, errShortKeyExists)
				require.Empty(t, r.key)
			}
		}
		require.Equal(t, 1, winners)
		requireMapping(t, mappings, "contended", winningURL)
	})
}

func TestCreateShortURLStopsOnGenerationError(t *testing.T) {
	forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
		generationErr := errors.New("random source unavailable")
		calls := 0
		mappings.generateKey = func() (string, error) {
			calls++
			return "partial", generationErr
		}
		key, err := mappings.Create(context.Background(), "https://example.com", "")
		require.ErrorIs(t, err, generationErr)
		require.Empty(t, key)
		require.Equal(t, 1, calls)
		requireNotFound(t, mappings, "partial")
	})
}

func TestCreateShortURLStopsOnStorageError(t *testing.T) {
	mappings := newTestMappings(t)
	failStorage(mappings)
	calls := 0
	mappings.generateKey = func() (string, error) { calls++; return "generated", nil }
	key, err := mappings.Create(context.Background(), "https://example.com", "")
	require.ErrorIs(t, err, errStorageDown)
	require.Empty(t, key)
	require.Equal(t, 1, calls)
}

func TestCreateShortURLStopsOnCanceledContext(t *testing.T) {
	forEachMappings(t, func(t *testing.T, mappings *shortURLMappings, _ func(time.Duration)) {
		mappings.generateKey = func() (string, error) { return "generated", nil }
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		key, err := mappings.Create(ctx, "https://example.com", "")
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, key)
		requireNotFound(t, mappings, "generated")
	})
}
