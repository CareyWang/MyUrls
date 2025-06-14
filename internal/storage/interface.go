package storage

import (
	"context"
	"time"
)

// Driver 定义存储驱动接口
type Driver interface {
	// Get 根据key获取值
	Get(ctx context.Context, key string) (string, error)

	// SetEx 设置key-value对，带过期时间
	SetEx(ctx context.Context, key string, value string, expiration time.Duration) error

	// Exists 检查key是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// TTL 获取key的剩余过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Expire 设置key的过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// Ping 检查存储连接是否正常
	Ping(ctx context.Context) error

	// Close 关闭存储连接
	Close() error
}
