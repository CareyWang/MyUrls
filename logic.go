package main

import (
	"context"
	"time"
)

// ShortToLong gets the long URL from a short URL
func ShortToLong(ctx context.Context, shortKey string) string {
	rc := GetRedisClient()
	return rc.Get(ctx, shortKey).Val()
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
