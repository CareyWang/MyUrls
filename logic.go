package main

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ShortToLong gets the long URL from a short URL.
// It returns an empty string when no valid mapping exists.
func ShortToLong(ctx context.Context, shortKey string) (string, error) {
	rc := GetRedisClient()
	longURL, err := rc.Get(ctx, shortKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return longURL, err
}

// Renew updates the expiration time of a short URL
func Renew(ctx context.Context, shortKey string, expiration time.Duration) error {
	rc := GetRedisClient()

	rs := rc.TTL(ctx, shortKey)
	if rs.Err() != nil {
		return rs.Err()
	}

	ttl := rs.Val()
	if ttl < 0 {
		return nil
	}

	return rc.Expire(ctx, shortKey, ttl+expiration).Err()
}
