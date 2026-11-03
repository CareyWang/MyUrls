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

// Resolve pipelines both reads, so it costs one round trip.
func (s *redisStore) Resolve(ctx context.Context, shortKey string) (string, time.Duration, error) {
	var longURL *redis.StringCmd
	var remaining *redis.DurationCmd
	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		longURL = pipe.Get(ctx, shortKey)
		remaining = pipe.PTTL(ctx, shortKey)
		return nil
	})
	if errors.Is(err, redis.Nil) {
		return "", 0, errShortURLNotFound
	}
	if err != nil {
		return "", 0, err
	}
	// go-redis passes Redis's reply codes through: -2 for a key that expired after
	// the GET, -1 for no expiration.
	if remaining.Val() == -2 {
		return "", 0, errShortURLNotFound
	}
	return longURL.Val(), remaining.Val(), nil
}

func (s *redisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close releases the connections of both clients.
func (s *redisStore) Close() error {
	return errors.Join(s.client.Close(), s.creationClient.Close())
}
