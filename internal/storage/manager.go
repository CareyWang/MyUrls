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

// GetLRUCache 获取存储驱动中的LRU缓存实例
func GetLRUCache() *LRUCache {
	if driver == nil {
		return nil
	}

	switch d := driver.(type) {
	case *RedisDriver:
		return d.GetLRUCache()
	case *SQLiteDriver:
		return d.GetLRUCache()
	default:
		return nil
	}
}

// ClearLRUCache 清空LRU缓存
func ClearLRUCache() {
	if driver == nil {
		return
	}

	switch d := driver.(type) {
	case *RedisDriver:
		d.ClearLRUCache()
	case *SQLiteDriver:
		d.ClearLRUCache()
	}
}
