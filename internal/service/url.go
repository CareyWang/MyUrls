package service

import (
	"context"
	"time"

	"github.com/CareyWang/MyUrls/internal/storage"
)

// ShortToLong gets the long URL from a short URL
func ShortToLong(ctx context.Context, shortKey string) string {
	driver := storage.GetDriver()
	result, err := driver.Get(ctx, shortKey)
	if err != nil {
		return ""
	}
	return result
}

// LongToShortOptions are the options for the LongToShort function
type LongToShortOptions struct {
	ShortKey   string
	URL        string
	Expiration time.Duration
}

// LongToShort creates a short URL from a long URL
func LongToShort(ctx context.Context, options *LongToShortOptions) error {
	driver := storage.GetDriver()
	return driver.SetEx(ctx, options.ShortKey, options.URL, options.Expiration)
}

// Renew updates the expiration time of a short URL
func Renew(ctx context.Context, shortKey string, expiration time.Duration) error {
	driver := storage.GetDriver()

	ttl, err := driver.TTL(ctx, shortKey)
	if err != nil {
		return err
	}

	if ttl < 0 {
		return nil
	}

	return driver.Expire(ctx, shortKey, ttl+expiration)
}

// CheckKeyExists checks if a key exists in storage
func CheckKeyExists(ctx context.Context, key string) (bool, error) {
	driver := storage.GetDriver()
	return driver.Exists(ctx, key)
}
