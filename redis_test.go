package main

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t testing.TB) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { client.Close() })
	return server, client
}

func newTestCreator(t testing.TB, client *redis.Client) *shortURLCreator {
	t.Helper()
	creator := newShortURLCreator(client.Options())
	t.Cleanup(func() {
		if err := creator.Close(); err != nil {
			t.Errorf("close short URL creator: %v", err)
		}
	})
	return creator
}
