package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	client *redis.Client
	cache  *LRUCache
}

func NewRedisDriver(addr, password string) (*RedisDriver, error) {
	const (
		defaultCacheSize = 128
		defaultCacheTTL  = 300 * time.Second
	)
	return NewRedisDriverWithCache(addr, password, defaultCacheSize, defaultCacheTTL)
}

func NewRedisDriverWithoutCache(addr, password string) (*RedisDriver, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &RedisDriver{
		client: client,
		cache:  nil,
	}, nil
}

func NewRedisDriverWithCache(addr, password string, cacheSize int, cacheTTL time.Duration) (*RedisDriver, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &RedisDriver{
		client: client,
		cache:  NewLRUCache(cacheSize, cacheTTL),
	}, nil
}

func (r *RedisDriver) Get(ctx context.Context, key string) (string, error) {
	if r.cache != nil {
		if val, ok := r.cache.Get(key); ok {
			return val, nil
		}
	}

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	if r.cache != nil {
		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			ttl = -1
		}
		r.cache.Set(key, val, ttl)
	}

	return val, nil
}

func (r *RedisDriver) SetEx(ctx context.Context, key string, value string, expiration time.Duration) error {
	if err := r.client.SetEx(ctx, key, value, expiration).Err(); err != nil {
		return err
	}

	if r.cache != nil {
		r.cache.Set(key, value, expiration)
	}
	return nil
}

func (r *RedisDriver) Exists(ctx context.Context, key string) (bool, error) {
	if r.cache != nil {
		if _, ok := r.cache.Get(key); ok {
			return true, nil
		}
	}
	res, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

func (r *RedisDriver) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

func (r *RedisDriver) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := r.client.Expire(ctx, key, expiration).Err(); err != nil {
		return err
	}

	if r.cache != nil {
		if val, ok := r.cache.Get(key); ok {
			r.cache.Set(key, val, expiration)
		}
	}

	return nil
}

func (r *RedisDriver) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisDriver) Close() error {
	if r.cache != nil {
		r.cache.Clear()
	}
	return r.client.Close()
}
