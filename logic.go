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

// LongToShortOptions are the options for the LongToShort function
type LongToShortOptions struct {
	ShortKey   string
	URL        string
	expiration time.Duration
}

// LongToShort creates a short URL from a long URL
func LongToShort(ctx context.Context, options *LongToShortOptions) error {
	rc := GetRedisClient()
	return rc.SetEx(ctx, options.ShortKey, options.URL, options.expiration).Err()
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

func CheckRedisKeyIfExist(ctx context.Context, key string) (bool, error) {
	rc := GetRedisClient()
	rs := rc.Exists(ctx, key)
	if rs.Err() != nil {
		return false, rs.Err()
	}

	return rs.Val() > 0, nil
}
