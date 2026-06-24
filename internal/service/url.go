package service

import (
	"context"
	"errors"
	"time"

	"github.com/CareyWang/MyUrls/internal/storage"
	"github.com/CareyWang/MyUrls/internal/utils"
)

// ErrShortKeyExists 表示请求的短链key已被占用（用户指定key冲突，或自动生成重试多次仍碰撞）
var ErrShortKeyExists = errors.New("short key already exists")

// maxAutoKeyAttempts 自动生成短链key时的最大重试次数
const maxAutoKeyAttempts = 5

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

// CreateShort 原子地创建一条短链映射，避免"先检查是否存在再写入"带来的竞态。
//
// shortKey 非空时，只尝试写入一次：若key已存在（且未过期）则返回 ErrShortKeyExists。
// shortKey 为空时，自动生成长度为 autoKeyLength 的随机key，碰撞时最多重试 maxAutoKeyAttempts 次。
func CreateShort(ctx context.Context, shortKey string, longURL string, ttl time.Duration, autoKeyLength int) (string, error) {
	driver := storage.GetDriver()

	if shortKey != "" {
		ok, err := driver.SetNXEx(ctx, shortKey, longURL, ttl)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", ErrShortKeyExists
		}
		return shortKey, nil
	}

	for i := 0; i < maxAutoKeyAttempts; i++ {
		candidate := utils.GenerateRandomString(autoKeyLength)
		ok, err := driver.SetNXEx(ctx, candidate, longURL, ttl)
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
	}

	return "", ErrShortKeyExists
}
