// FILEPATH: /root/CareyWang/MyUrls/logic_test.go

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLongToShortAndShortToLong(t *testing.T) {
	ctx := context.Background()
	initRedisClient(mockRedisOptions)

	shortKey := "testKey"
	longURL := "https://example.com"

	err := LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        longURL,
		expiration: 60 * time.Second,
	})
	assert.NoError(t, err)
	// delete test data from redis
	defer GetRedisClient().Del(ctx, shortKey)

	resultLongURL := ShortToLong(ctx, shortKey)
	assert.Equal(t, longURL, resultLongURL)
}
