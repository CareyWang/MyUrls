package main

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisStore keeps mappings in Redis, with a client per retry policy.
type redisStore struct {
	client         *redis.Client
	creationClient *redis.Client
}

func newRedisStore(addr, password string) *redisStore {
	return &redisStore{
		client: redis.NewClient(&redis.Options{Addr: addr, Password: password}),
		// Replaying SET NX after a lost reply can turn our own write into a collision.
		creationClient: redis.NewClient(&redis.Options{Addr: addr, Password: password, MaxRetries: -1}),
	}
}

// Claim sets the value and expiration atomically, without overwriting a valid mapping.
func (s *redisStore) Claim(ctx context.Context, shortKey, longURL string, ttl time.Duration) (bool, error) {
	return s.creationClient.SetNX(ctx, shortKey, longURL, ttl).Result()
}

func (s *redisStore) Resolve(ctx context.Context, shortKey string) (string, error) {
	longURL, err := s.client.Get(ctx, shortKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", errShortURLNotFound
	}
	return longURL, err
}

func (s *redisStore) TTL(ctx context.Context, shortKey string) (time.Duration, error) {
	remaining, err := s.client.PTTL(ctx, shortKey).Result()
	if err != nil {
		return 0, err
	}
	// go-redis passes Redis's reply codes through: -2 for a missing key, -1 for no expiration.
	if remaining == -2 {
		return 0, errShortURLNotFound
	}
	return remaining, nil
}

func (s *redisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close releases the connections of both clients.
func (s *redisStore) Close() error {
	return errors.Join(s.client.Close(), s.creationClient.Close())
}
