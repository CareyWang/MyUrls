package storage

import (
	"fmt"
	"time"

	"github.com/CareyWang/MyUrls/internal/config"
)

var driver Driver

// InitStorage 初始化存储驱动
func InitStorage(storageConfig *config.StorageConfig) error {
	var storageDriver Driver
	var err error

	switch storageConfig.Type {
	case config.StorageRedis:
		if storageConfig.CacheEnabled {
			storageDriver, err = NewRedisDriverWithCache(
				storageConfig.RedisAddr,
				storageConfig.RedisPassword,
				storageConfig.CacheSize,
				time.Duration(storageConfig.CacheTTL)*time.Second,
			)
		} else {
			storageDriver, err = NewRedisDriverWithoutCache(
				storageConfig.RedisAddr,
				storageConfig.RedisPassword,
			)
		}
	case config.StorageSQLite:
		if storageConfig.CacheEnabled {
			storageDriver, err = NewSQLiteDriverWithCache(
				storageConfig.SQLiteFile,
				storageConfig.CacheSize,
				time.Duration(storageConfig.CacheTTL)*time.Second,
			)
		} else {
			storageDriver, err = NewSQLiteDriverWithoutCache(
				storageConfig.SQLiteFile,
			)
		}
	default:
		return fmt.Errorf("unsupported storage type: %s", storageConfig.Type)
	}

	if err != nil {
		return err
	}

	driver = storageDriver
	return nil
}

// GetDriver 获取存储驱动
func GetDriver() Driver {
	return driver
}
