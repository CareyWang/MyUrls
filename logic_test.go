package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreatedShortURLResolves(t *testing.T) {
	_, client := newTestRedis(t)
	previousClient := RedisClient
	RedisClient = client
	t.Cleanup(func() { RedisClient = previousClient })

	creator := newTestCreator(t, client)
	key, err := creator.Create(context.Background(), "https://example.com", "testKey")
	require.NoError(t, err)
	longURL, err := ShortToLong(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, "https://example.com", longURL)
	longURL, err = ShortToLong(context.Background(), "missing")
	require.NoError(t, err)
	require.Empty(t, longURL)
}
