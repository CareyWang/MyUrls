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
	require.Equal(t, "https://example.com", ShortToLong(context.Background(), key))
	require.Empty(t, ShortToLong(context.Background(), "missing"))
}
