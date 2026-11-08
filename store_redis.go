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
func (s *redisStore) Claim(ctx context.Context, shortKey, longURL string, life lifetime) (bool, error) {
	if !life.valid() {
		return false, errInvalidLifetime
	}
	// A zero expiration tells Redis to keep the key forever.
	var ttl time.Duration
	if !life.isPermanent() {
		ttl = life.remaining
	}
	return s.creationClient.SetNX(ctx, shortKey, longURL, ttl).Result()
}

// Resolve pipelines both reads, so it costs one round trip.
func (s *redisStore) Resolve(ctx context.Context, shortKey string) (string, lifetime, error) {
	var longURL *redis.StringCmd
	var remaining *redis.DurationCmd
	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		longURL = pipe.Get(ctx, shortKey)
		remaining = pipe.PTTL(ctx, shortKey)
		return nil
	})
	if errors.Is(err, redis.Nil) {
		return "", lifetime{}, errShortURLNotFound
	}
	if err != nil {
		return "", lifetime{}, err
	}
	// go-redis passes Redis's reply codes through: -2 for a key that expired after
	// the GET, -1 for no expiration, and 0 for under a millisecond left, which
	// counts as expired.
	switch ttl := remaining.Val(); {
	case ttl == -1:
		return longURL.Val(), permanentLifetime, nil
	case ttl <= 0:
		return "", lifetime{}, errShortURLNotFound
	default:
		return longURL.Val(), remainingLifetime(ttl), nil
	}
}

func (s *redisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close releases the connections of both clients.
func (s *redisStore) Close() error {
	return errors.Join(s.client.Close(), s.creationClient.Close())
}
