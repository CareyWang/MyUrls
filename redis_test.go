package main

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var mockRedisOptions = &redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
}

func TestGetRedisClient(t *testing.T) {
	client := GetRedisClient()
	assert.Nil(t, client)

	initRedisClient(mockRedisOptions)
	client = GetRedisClient()
	assert.NotNil(t, client)

	// Test redis exec commands and response
	ctx := context.Background()
	rs := client.Ping(ctx)
	assert.Nil(t, rs.Err())
	assert.Equal(t, "PONG", rs.Val())

	rsCmd := GetRedisClient().Do(ctx, "dbsize")
	assert.Nil(t, rsCmd.Err())
}

func BenchmarkGetRedisClient(b *testing.B) {
	initRedisClient(mockRedisOptions)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetRedisClient().Get(context.Background(), "key")
	}
}
