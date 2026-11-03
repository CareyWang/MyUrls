package main

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"
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

// mappingStore is a storage backend for mappings.
type mappingStore interface {
	// Claim stores a mapping unless a valid one holds the short key, and reports
	// whether it did. A zero ttl stores a mapping that never expires.
	// An uncertain write must return an error, not false.
	Claim(ctx context.Context, shortKey, longURL string, ttl time.Duration) (bool, error)
	// Resolve returns errShortURLNotFound when no valid mapping exists.
	Resolve(ctx context.Context, shortKey string) (string, error)
	Ping(ctx context.Context) error
	Close() error
}

// shortURLMappings stores the mappings from short keys to long URLs.
type shortURLMappings struct {
	store       mappingStore
	generateKey func() (string, error)
}

// newShortURLMappings returns mappings that callers must close.
func newShortURLMappings(store mappingStore) *shortURLMappings {
	return &shortURLMappings{store: store, generateKey: generateShortKey}
}

// Close releases the storage.
func (m *shortURLMappings) Close() error {
	return m.store.Close()
}

// Ping checks that the mapping storage is reachable.
func (m *shortURLMappings) Ping(ctx context.Context) error {
	return m.store.Ping(ctx)
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

		created, err := m.store.Claim(ctx, shortKey, longURL, defaultTTL)
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
	return m.store.Resolve(ctx, shortKey)
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
