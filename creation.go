package main

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultTTL            = time.Hour * 24 * 365
	defaultShortKeyLength = 7
	maxShortKeyRetries    = 3 // Retries after the initial attempt.
	letterBytes           = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	errShortKeyExists           = errors.New("short key already exists")
	errShortKeyRetriesExhausted = errors.New("short key generation retries exhausted")
)

type shortURLCreator struct {
	client      *redis.Client
	generateKey func() (string, error)
}

// newShortURLCreator owns a separate client that callers must close.
func newShortURLCreator(options *redis.Options) *shortURLCreator {
	creationOptions := *options
	// Replaying SET NX after a lost reply can turn our own write into a collision.
	creationOptions.MaxRetries = -1
	return &shortURLCreator{client: redis.NewClient(&creationOptions), generateKey: generateShortKey}
}

// Close releases the creation client's connections.
func (c *shortURLCreator) Close() error {
	return c.client.Close()
}

// Create stores a long URL under an unused short key for one year.
// An empty shortKey generates a key, with at most three retries on collision.
// Generation and storage errors stop creation without further retries.
func (c *shortURLCreator) Create(ctx context.Context, longURL, shortKey string) (string, error) {
	generated := shortKey == ""
	for attempt := 0; attempt <= maxShortKeyRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if generated {
			var err error
			shortKey, err = c.generateKey()
			if err != nil {
				return "", err
			}
		}

		// Claim the key and set its expiration atomically, without overwriting it.
		created, err := c.client.SetNX(ctx, shortKey, longURL, defaultTTL).Result()
		if err != nil {
			return "", err
		}
		if created {
			return shortKey, nil
		}
		if !generated {
			return "", errShortKeyExists
		}
	}
	return "", errShortKeyRetriesExhausted
}

func generateShortKey() (string, error) {
	b := make([]byte, defaultShortKeyLength)
	alphabetSize := big.NewInt(int64(len(letterBytes)))
	for i := range b {
		n, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[n.Int64()]
	}
	return string(b), nil
}
