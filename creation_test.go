package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestCreateShortURLExpiration(t *testing.T) {
	for _, requestedKey := range []string{"custom", ""} {
		t.Run(fmt.Sprintf("key=%q", requestedKey), func(t *testing.T) {
			server, client := newTestRedis(t)
			creator := newTestCreator(t, client)
			ctx := context.Background()
			key, err := creator.Create(ctx, "https://example.com/original", requestedKey)
			require.NoError(t, err)
			if requestedKey == "" {
				require.Regexp(t, `^[0-9a-zA-Z]{7}$`, key)
			} else {
				require.Equal(t, requestedKey, key)
			}
			require.Equal(t, "https://example.com/original", client.Get(ctx, key).Val())
			require.Equal(t, 365*24*time.Hour, client.TTL(ctx, key).Val())

			server.FastForward(time.Hour)
			_, err = creator.Create(ctx, "https://example.com/replacement", key)
			require.ErrorIs(t, err, errShortKeyExists)
			require.Equal(t, "https://example.com/original", client.Get(ctx, key).Val())
			require.Equal(t, 365*24*time.Hour-time.Hour, client.TTL(ctx, key).Val())

			server.FastForward(365*24*time.Hour - time.Hour)
			require.ErrorIs(t, client.Get(ctx, key).Err(), redis.Nil)
			_, err = creator.Create(ctx, "https://example.com/replacement", key)
			require.NoError(t, err)
			require.Equal(t, "https://example.com/replacement", client.Get(ctx, key).Val())
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
			_, client := newTestRedis(t)
			ctx := context.Background()
			require.NoError(t, client.Set(ctx, "taken", "https://example.com/original", time.Hour).Err())
			creator := newTestCreator(t, client)
			calls := 0
			creator.generateKey = func() (string, error) {
				calls++
				if calls <= tc.collisions {
					return "taken", nil
				}
				return "fresh", nil
			}

			key, err := creator.Create(ctx, "https://example.com/new", "")
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.wantCalls, calls)
			require.Equal(t, "https://example.com/original", client.Get(ctx, "taken").Val())
			require.Equal(t, time.Hour, client.TTL(ctx, "taken").Val())
			if tc.wantErr != nil {
				require.Empty(t, key)
				require.EqualValues(t, 1, client.DBSize(ctx).Val())
			} else {
				require.Equal(t, "fresh", key)
				require.Equal(t, "https://example.com/new", client.Get(ctx, key).Val())
			}
		})
	}
}

func TestCreateShortURLCustomCollisionDoesNotGenerate(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "custom", "original", 0).Err())
	creator := newTestCreator(t, client)
	creator.generateKey = func() (string, error) {
		t.Fatal("a custom key must not trigger automatic generation")
		return "", nil
	}
	key, err := creator.Create(ctx, "replacement", "custom")
	require.ErrorIs(t, err, errShortKeyExists)
	require.Empty(t, key)
	require.Equal(t, "original", client.Get(ctx, "custom").Val())
}

func TestCreateShortURLConcurrentClaims(t *testing.T) {
	_, client := newTestRedis(t)
	creator := newTestCreator(t, client)
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
			key, err := creator.Create(ctx, url, "contended")
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
	require.Equal(t, winningURL, client.Get(ctx, "contended").Val())
}

func TestCreateShortURLStopsOnGenerationError(t *testing.T) {
	_, client := newTestRedis(t)
	creator := newTestCreator(t, client)
	generationErr := errors.New("random source unavailable")
	calls := 0
	creator.generateKey = func() (string, error) {
		calls++
		return "partial", generationErr
	}
	key, err := creator.Create(context.Background(), "https://example.com", "")
	require.ErrorIs(t, err, generationErr)
	require.Empty(t, key)
	require.Equal(t, 1, calls)
	require.Zero(t, client.DBSize(context.Background()).Val())
}

func TestCreateShortURLStopsOnStorageError(t *testing.T) {
	server, client := newTestRedis(t)
	server.SetError("ERR storage unavailable")
	creator := newTestCreator(t, client)
	calls := 0
	creator.generateKey = func() (string, error) { calls++; return "generated", nil }
	key, err := creator.Create(context.Background(), "https://example.com", "")
	require.EqualError(t, err, "ERR storage unavailable")
	require.Empty(t, key)
	require.Equal(t, 1, calls)
}

func TestCreateShortURLStopsOnCanceledContext(t *testing.T) {
	_, client := newTestRedis(t)
	creator := newTestCreator(t, client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	key, err := creator.Create(ctx, "https://example.com", "")
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, key)
	require.Zero(t, client.DBSize(context.Background()).Val())
}
