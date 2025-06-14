package config

import (
	"os"
	"strings"
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
}

func GetStorageConfig() *StorageConfig {
	config := &StorageConfig{
		Type:          StorageRedis, // 默认为Redis
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		SQLiteFile:    "./data/myurls.db",
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

	return config
}
