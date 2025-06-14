package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	client *redis.Client
}

func NewRedisDriver(addr, password string) (*RedisDriver, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &RedisDriver{client: client}, nil
}

func (r *RedisDriver) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisDriver) SetEx(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.SetEx(ctx, key, value, expiration).Err()
}

func (r *RedisDriver) Exists(ctx context.Context, key string) (bool, error) {
	result := r.client.Exists(ctx, key)
	if result.Err() != nil {
		return false, result.Err()
	}
	return result.Val() > 0, nil
}

func (r *RedisDriver) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

func (r *RedisDriver) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

func (r *RedisDriver) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisDriver) Close() error {
	return r.client.Close()
}
