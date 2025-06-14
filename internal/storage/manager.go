package storage

import (
	"fmt"

	"github.com/CareyWang/MyUrls/internal/config"
)

var driver Driver

// InitStorage 初始化存储驱动
func InitStorage(storageConfig *config.StorageConfig) error {
	var storageDriver Driver
	var err error

	switch storageConfig.Type {
	case config.StorageRedis:
		storageDriver, err = NewRedisDriver(storageConfig.RedisAddr, storageConfig.RedisPassword)
	case config.StorageSQLite:
		storageDriver, err = NewSQLiteDriver(storageConfig.SQLiteFile)
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
