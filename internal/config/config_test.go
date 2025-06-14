package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStorageConfig_Defaults(t *testing.T) {
	// 清除环境变量
	os.Unsetenv("MYURLS_STORAGE_TYPE")
	os.Unsetenv("MYURLS_REDIS_CONN")
	os.Unsetenv("MYURLS_REDIS_PASSWORD")
	os.Unsetenv("MYURLS_SQLITE_FILE")

	config := GetStorageConfig()

	assert.Equal(t, StorageRedis, config.Type)
	assert.Equal(t, "localhost:6379", config.RedisAddr)
	assert.Equal(t, "", config.RedisPassword)
	assert.Equal(t, "./data/myurls.db", config.SQLiteFile)
}

func TestGetStorageConfig_EnvironmentVariables(t *testing.T) {
	// 设置环境变量
	os.Setenv("MYURLS_STORAGE_TYPE", "sqlite")
	os.Setenv("MYURLS_REDIS_CONN", "redis:6379")
	os.Setenv("MYURLS_REDIS_PASSWORD", "secret")
	os.Setenv("MYURLS_SQLITE_FILE", "/custom/path.db")

	defer func() {
		// 清理环境变量
		os.Unsetenv("MYURLS_STORAGE_TYPE")
		os.Unsetenv("MYURLS_REDIS_CONN")
		os.Unsetenv("MYURLS_REDIS_PASSWORD")
		os.Unsetenv("MYURLS_SQLITE_FILE")
	}()

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
		os.Setenv("MYURLS_STORAGE_TYPE", testCase)
		config := GetStorageConfig()
		assert.Equal(t, expected[i], config.Type, "Failed for input: %s", testCase)
	}

	os.Unsetenv("MYURLS_STORAGE_TYPE")
}
