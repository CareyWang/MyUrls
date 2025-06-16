package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStorageConfig_Defaults(t *testing.T) {
	// 清除所有相关环境变量
	envVars := []string{
		"MYURLS_STORAGE_TYPE",
		"MYURLS_STORAGE_REDIS_ADDR",
		"MYURLS_STORAGE_REDIS_PASSWORD",
		"MYURLS_STORAGE_SQLITE_FILE",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}

	// 清理全局配置以确保使用默认值
	globalConfig = nil

	config := GetStorageConfig()

	assert.Equal(t, StorageRedis, config.Type)
	assert.Equal(t, "localhost:6379", config.RedisAddr)
	assert.Equal(t, "", config.RedisPassword)
	assert.Equal(t, "./data/myurls.db", config.SQLiteFile)
}

func TestGetStorageConfig_EnvironmentVariables(t *testing.T) {
	// 设置环境变量 - 使用新的变量名
	os.Setenv("MYURLS_STORAGE_TYPE", "sqlite")
	os.Setenv("MYURLS_STORAGE_REDIS_ADDR", "redis:6379")
	os.Setenv("MYURLS_STORAGE_REDIS_PASSWORD", "secret")
	os.Setenv("MYURLS_STORAGE_SQLITE_FILE", "/custom/path.db")

	defer func() {
		// 清理环境变量
		os.Unsetenv("MYURLS_STORAGE_TYPE")
		os.Unsetenv("MYURLS_STORAGE_REDIS_ADDR")
		os.Unsetenv("MYURLS_STORAGE_REDIS_PASSWORD")
		os.Unsetenv("MYURLS_STORAGE_SQLITE_FILE")
	}()

	// 清理全局配置以确保重新加载
	globalConfig = nil

	config := GetStorageConfig()

	assert.Equal(t, StorageSQLite, config.Type)
	assert.Equal(t, "redis:6379", config.RedisAddr)
	assert.Equal(t, "secret", config.RedisPassword)
	assert.Equal(t, "/custom/path.db", config.SQLiteFile)
}

func TestGetStorageConfig_CaseInsensitive(t *testing.T) {
	// 测试存储类型的大小写不敏感
	testCases := []string{"REDIS", "Redis", "redis", "SQLITE", "SQLite", "sqlite"}
	expected := []StorageType{StorageRedis, StorageRedis, StorageRedis, StorageSQLite, StorageSQLite, StorageSQLite}

	for i, testCase := range testCases {
		// 清理全局配置
		globalConfig = nil

		os.Setenv("MYURLS_STORAGE_TYPE", testCase)
		config := GetStorageConfig()
		assert.Equal(t, expected[i], config.Type, "Failed for input: %s", testCase)
	}

	os.Unsetenv("MYURLS_STORAGE_TYPE")
	// 清理全局配置
	globalConfig = nil
}

func TestGetStorageConfig_CacheEnabled(t *testing.T) {
	// 测试缓存启用
	os.Setenv("MYURLS_STORAGE_CACHE_ENABLED", "true")
	globalConfig = nil
	config := GetStorageConfig()
	assert.True(t, config.CacheEnabled)
	os.Unsetenv("MYURLS_STORAGE_CACHE_ENABLED")

	// 测试缓存禁用
	os.Setenv("MYURLS_STORAGE_CACHE_ENABLED", "false")
	globalConfig = nil
	config = GetStorageConfig()
	assert.False(t, config.CacheEnabled)
	os.Unsetenv("MYURLS_STORAGE_CACHE_ENABLED")

	// 测试无效值（应该使用默认值true）
	os.Setenv("MYURLS_STORAGE_CACHE_ENABLED", "invalid")
	globalConfig = nil
	config = GetStorageConfig()
	assert.True(t, config.CacheEnabled) // 应该使用默认值
	os.Unsetenv("MYURLS_STORAGE_CACHE_ENABLED")

	// 清理全局配置
	globalConfig = nil
}
