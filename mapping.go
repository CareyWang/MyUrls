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
	errShortURLNotFound         = errors.New("short URL not found")
)

// shortURLMappings stores the mappings from short keys to long URLs.
type shortURLMappings struct {
	client         *redis.Client
	creationClient *redis.Client
	generateKey    func() (string, error)
}

// newShortURLMappings returns mappings that callers must close.
func newShortURLMappings(addr, password string) *shortURLMappings {
	return &shortURLMappings{
		client: redis.NewClient(&redis.Options{Addr: addr, Password: password}),
		// Replaying SET NX after a lost reply can turn our own write into a collision.
		creationClient: redis.NewClient(&redis.Options{Addr: addr, Password: password, MaxRetries: -1}),
		generateKey:    generateShortKey,
	}
}

// Close releases the connections of both clients.
func (m *shortURLMappings) Close() error {
	return errors.Join(m.client.Close(), m.creationClient.Close())
}

// Ping checks that the mapping storage is reachable.
func (m *shortURLMappings) Ping(ctx context.Context) error {
	return m.client.Ping(ctx).Err()
}

// Create stores a long URL under an unused short key for one year.
// An empty shortKey generates a key, with at most three retries on collision.
// Generation and storage errors stop creation without further retries.
func (m *shortURLMappings) Create(ctx context.Context, longURL, shortKey string) (string, error) {
	generated := shortKey == ""
	for attempt := 0; attempt <= maxShortKeyRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if generated {
			var err error
			shortKey, err = m.generateKey()
			if err != nil {
				return "", err
			}
		}

		// Claim the key and set its expiration atomically, without overwriting it.
		created, err := m.creationClient.SetNX(ctx, shortKey, longURL, defaultTTL).Result()
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

// Resolve returns the long URL of a short key.
// It returns errShortURLNotFound when no valid mapping exists.
func (m *shortURLMappings) Resolve(ctx context.Context, shortKey string) (string, error) {
	longURL, err := m.client.Get(ctx, shortKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", errShortURLNotFound
	}
	return longURL, err
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
