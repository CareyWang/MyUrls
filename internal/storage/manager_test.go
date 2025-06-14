package storage

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CareyWang/MyUrls/internal/config"
)

func TestInitStorage_Redis(t *testing.T) {
	// 跳过Redis测试如果没有Redis服务器
	if os.Getenv("SKIP_REDIS_TESTS") != "" {
		t.Skip("Skipping Redis tests")
	}

	storageConfig := &config.StorageConfig{
		Type:          config.StorageRedis,
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
	}

	err := InitStorage(storageConfig)
	assert.NoError(t, err)

	storageDriver := GetDriver()
	assert.NotNil(t, storageDriver)

	// 验证是RedisDriver类型
	_, ok := storageDriver.(*RedisDriver)
	assert.True(t, ok)

	// 清理
	storageDriver.Close()
}

func TestInitStorage_SQLite(t *testing.T) {
	tmpFile := "./test_manager.db"
	defer os.Remove(tmpFile)

	storageConfig := &config.StorageConfig{
		Type:       config.StorageSQLite,
		SQLiteFile: tmpFile,
	}

	err := InitStorage(storageConfig)
	assert.NoError(t, err)

	storageDriver := GetDriver()
	assert.NotNil(t, storageDriver)

	// 验证是SQLiteDriver类型
	_, ok := storageDriver.(*SQLiteDriver)
	assert.True(t, ok)

	// 清理
	storageDriver.Close()
}

func TestInitStorage_UnsupportedType(t *testing.T) {
	storageConfig := &config.StorageConfig{
		Type: config.StorageType("unsupported"),
	}

	err := InitStorage(storageConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported storage type")
}

func TestInitStorage_InvalidRedisConfig(t *testing.T) {
	storageConfig := &config.StorageConfig{
		Type:      config.StorageRedis,
		RedisAddr: "invalid:host:port",
	}

	// Redis驱动创建可能不会立即失败，这取决于具体实现
	// 但至少不应该panic
	assert.NotPanics(t, func() {
		InitStorage(storageConfig)
	})
}

func TestInitStorage_InvalidSQLiteConfig(t *testing.T) {
	storageConfig := &config.StorageConfig{
		Type:       config.StorageSQLite,
		SQLiteFile: "/invalid/path/that/does/not/exist/file.db",
	}

	err := InitStorage(storageConfig)
	// SQLite可能会创建目录或返回错误，这取决于实现
	// 这里主要测试不会panic
	assert.NotPanics(t, func() {
		InitStorage(storageConfig)
	})

	// 如果返回错误，应该是有意义的
	if err != nil {
		assert.NotEmpty(t, err.Error())
	}
}

func TestGetDriver_BeforeInit(t *testing.T) {
	// 保存原始驱动
	originalDriver := driver
	defer func() {
		driver = originalDriver
	}()

	// 清空驱动
	driver = nil

	storageDriver := GetDriver()
	assert.Nil(t, storageDriver)
}

func TestStorageManager_Integration(t *testing.T) {
	tmpFile := "./test_integration.db"
	defer os.Remove(tmpFile)

	// 测试完整的初始化流程
	storageConfig := config.GetStorageConfig()
	storageConfig.Type = config.StorageSQLite
	storageConfig.SQLiteFile = tmpFile

	err := InitStorage(storageConfig)
	require.NoError(t, err)

	storageDriver := GetDriver()
	require.NotNil(t, storageDriver)

	// 测试驱动功能
	ctx := context.Background()
	err = storageDriver.Ping(ctx)
	assert.NoError(t, err)

	// 清理
	storageDriver.Close()
}
