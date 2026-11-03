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

func newTestMappingsAt(t testing.TB, addr string) *shortURLMappings {
	t.Helper()
	mappings := newShortURLMappings(newRedisStore(addr, ""), nil)
	t.Cleanup(func() {
		if err := mappings.Close(); err != nil {
			t.Errorf("close short URL mappings: %v", err)
		}
	})
	return mappings
}

func newTestMappings(t testing.TB) (*miniredis.Miniredis, *shortURLMappings) {
	t.Helper()
	server := miniredis.RunT(t)
	return server, newTestMappingsAt(t, server.Addr())
}

func requireMapping(t testing.TB, mappings *shortURLMappings, shortKey, wantLongURL string) {
	t.Helper()
	longURL, err := mappings.Resolve(context.Background(), shortKey)
	require.NoError(t, err)
	require.Equal(t, wantLongURL, longURL)
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

	_, mappings := newTestMappings(t)
	require.NoError(t, mappings.Ping(context.Background()))
	_, err := mappings.Create(context.Background(), "https://example.com", "custom")
	require.NoError(t, err)

	captured.mu.Lock()
	defer captured.mu.Unlock()
	require.NotContains(t, captured.logs.String(), "failed to initialize maintnotifications")
}

func TestPingShortURLMappings(t *testing.T) {
	server, mappings := newTestMappings(t)
	ctx := context.Background()
	require.NoError(t, mappings.Ping(ctx))

	server.SetError("ERR storage unavailable")
	require.EqualError(t, mappings.Ping(ctx), "ERR storage unavailable")
}

func TestResolveShortURL(t *testing.T) {
	server, mappings := newTestMappings(t)
	ctx := context.Background()
	key, err := mappings.Create(ctx, "https://example.com", "testKey")
	require.NoError(t, err)
	requireMapping(t, mappings, key, "https://example.com")

	longURL, err := mappings.Resolve(ctx, "missing")
	require.ErrorIs(t, err, errShortURLNotFound)
	require.Empty(t, longURL)

	server.SetError("ERR storage unavailable")
	longURL, err = mappings.Resolve(ctx, key)
	require.EqualError(t, err, "ERR storage unavailable")
	require.NotErrorIs(t, err, errShortURLNotFound)
	require.Empty(t, longURL)
}

func TestCreateShortURLExpiration(t *testing.T) {
	for _, requestedKey := range []string{"custom", ""} {
		t.Run(fmt.Sprintf("key=%q", requestedKey), func(t *testing.T) {
			server, mappings := newTestMappings(t)
			ctx := context.Background()
			key, err := mappings.Create(ctx, "https://example.com/original", requestedKey)
			require.NoError(t, err)
			if requestedKey == "" {
				require.Regexp(t, `^[0-9a-zA-Z]{7}$`, key)
			} else {
				require.Equal(t, requestedKey, key)
			}
			requireMapping(t, mappings, key, "https://example.com/original")
			require.Equal(t, 365*24*time.Hour, server.TTL(key))

			server.FastForward(time.Hour)
			_, err = mappings.Create(ctx, "https://example.com/replacement", key)
			require.ErrorIs(t, err, errShortKeyExists)
			requireMapping(t, mappings, key, "https://example.com/original")
			require.Equal(t, 365*24*time.Hour-time.Hour, server.TTL(key))

			server.FastForward(365*24*time.Hour - time.Hour)
			_, err = mappings.Resolve(ctx, key)
			require.ErrorIs(t, err, errShortURLNotFound)
			_, err = mappings.Create(ctx, "https://example.com/replacement", key)
			require.NoError(t, err)
			requireMapping(t, mappings, key, "https://example.com/replacement")
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
			server, mappings := newTestMappings(t)
			ctx := context.Background()
			require.NoError(t, server.Set("taken", "https://example.com/original"))
			server.SetTTL("taken", time.Hour)
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
			require.Equal(t, time.Hour, server.TTL("taken"))
			if tc.wantErr != nil {
				require.Empty(t, key)
				require.Len(t, server.Keys(), 1)
			} else {
				require.Equal(t, "fresh", key)
				requireMapping(t, mappings, key, "https://example.com/new")
			}
		})
	}
}

func TestCreateShortURLCustomCollisionDoesNotGenerate(t *testing.T) {
	server, mappings := newTestMappings(t)
	ctx := context.Background()
	require.NoError(t, server.Set("custom", "original"))
	mappings.generateKey = func() (string, error) {
		t.Fatal("a custom key must not trigger automatic generation")
		return "", nil
	}
	key, err := mappings.Create(ctx, "replacement", "custom")
	require.ErrorIs(t, err, errShortKeyExists)
	require.Empty(t, key)
	requireMapping(t, mappings, "custom", "original")
}

func TestCreateShortURLConcurrentClaims(t *testing.T) {
	_, mappings := newTestMappings(t)
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
}

func TestCreateShortURLStopsOnGenerationError(t *testing.T) {
	server, mappings := newTestMappings(t)
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
	require.Empty(t, server.Keys())
}

func TestCreateShortURLStopsOnStorageError(t *testing.T) {
	server, mappings := newTestMappings(t)
	server.SetError("ERR storage unavailable")
	calls := 0
	mappings.generateKey = func() (string, error) { calls++; return "generated", nil }
	key, err := mappings.Create(context.Background(), "https://example.com", "")
	require.EqualError(t, err, "ERR storage unavailable")
	require.Empty(t, key)
	require.Equal(t, 1, calls)
}

func TestCreateShortURLStopsOnCanceledContext(t *testing.T) {
	server, mappings := newTestMappings(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	key, err := mappings.Create(ctx, "https://example.com", "")
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, key)
	require.Empty(t, server.Keys())
}
