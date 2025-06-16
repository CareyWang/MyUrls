package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type StorageType string

const (
	StorageRedis  StorageType = "redis"
	StorageSQLite StorageType = "sqlite"
)

type StorageConfig struct {
	Type          StorageType
	RedisAddr     string
	RedisPassword string
	SQLiteFile    string
	CacheSize     int
	CacheTTL      time.Duration
}

func GetStorageConfig() *StorageConfig {
	config := &StorageConfig{
		Type:          StorageRedis, // 默认为Redis
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		SQLiteFile:    "./data/myurls.db",
		CacheSize:     1024,
		CacheTTL:      5 * time.Minute,
	}

	// 从环境变量读取配置
	if storageType := os.Getenv("MYURLS_STORAGE_TYPE"); storageType != "" {
		config.Type = StorageType(strings.ToLower(storageType))
	}

	if redisAddr := os.Getenv("MYURLS_REDIS_CONN"); redisAddr != "" {
		config.RedisAddr = redisAddr
	}

	if redisPassword := os.Getenv("MYURLS_REDIS_PASSWORD"); redisPassword != "" {
		config.RedisPassword = redisPassword
	}

	if sqliteFile := os.Getenv("MYURLS_SQLITE_FILE"); sqliteFile != "" {
		config.SQLiteFile = sqliteFile
	}

	// 从环境变量中读取缓存配置
	if cacheSizeStr := os.Getenv("MYURLS_CACHE_SIZE"); cacheSizeStr != "" {
		if size, err := strconv.Atoi(cacheSizeStr); err == nil && size > 0 {
			config.CacheSize = size
		}
	}
	if cacheTTLStr := os.Getenv("MYURLS_CACHE_TTL"); cacheTTLStr != "" {
		if ttl, err := time.ParseDuration(cacheTTLStr); err == nil && ttl > 0 {
			config.CacheTTL = ttl
		}
	}

	return config
}
