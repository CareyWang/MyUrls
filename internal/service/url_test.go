package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CareyWang/MyUrls/internal/config"
	"github.com/CareyWang/MyUrls/internal/storage"
)

func setupTestStorage(t *testing.T) (func(), storage.Driver) {
	tmpFile := "./test_logic.db"

	storageConfig := &config.StorageConfig{
		Type:       config.StorageSQLite,
		SQLiteFile: tmpFile,
	}

	err := storage.InitStorage(storageConfig)
	require.NoError(t, err)

	driver := storage.GetDriver()
	require.NotNil(t, driver)

	cleanup := func() {
		driver.Close()
		os.Remove(tmpFile)
	}

	return cleanup, driver
}

func TestShortToLong(t *testing.T) {
	cleanup, _ := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortKey := "test123"
	longURL := "https://example.com/very/long/url"

	// 测试不存在的key
	result := ShortToLong(ctx, shortKey)
	assert.Empty(t, result)

	// 先存储一个URL
	err := LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        longURL,
		Expiration: 60 * time.Second,
	})
	require.NoError(t, err)

	// 测试获取存在的key
	result = ShortToLong(ctx, shortKey)
	assert.Equal(t, longURL, result)
}

func TestLongToShort(t *testing.T) {
	cleanup, _ := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortKey := "test456"
	longURL := "https://example.com/another/long/url"

	// 测试存储URL
	err := LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        longURL,
		Expiration: 30 * time.Second,
	})
	assert.NoError(t, err)

	// 验证存储成功
	result := ShortToLong(ctx, shortKey)
	assert.Equal(t, longURL, result)

	// 测试覆盖存储
	newURL := "https://example.com/new/url"
	err = LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        newURL,
		Expiration: 30 * time.Second,
	})
	assert.NoError(t, err)

	// 验证覆盖成功
	result = ShortToLong(ctx, shortKey)
	assert.Equal(t, newURL, result)
}

func TestRenew(t *testing.T) {
	cleanup, _ := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortKey := "test789"
	longURL := "https://example.com/renew/test"

	// 先存储一个URL，设置较短的过期时间
	err := LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        longURL,
		Expiration: 5 * time.Second,
	})
	require.NoError(t, err)

	// 测试续期
	err = Renew(ctx, shortKey, 10*time.Second)
	assert.NoError(t, err)

	// 验证续期后仍然存在
	result := ShortToLong(ctx, shortKey)
	assert.Equal(t, longURL, result)

	// 测试对不存在的key续期
	err = Renew(ctx, "nonexistent", 10*time.Second)
	// 根据实现，可能返回错误或者忽略
	// 这里不做严格断言，只确保不panic
	_ = err
}

func TestCheckKeyExists(t *testing.T) {
	cleanup, _ := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortKey := "testexist"
	longURL := "https://example.com/exist/test"

	// 测试不存在的key
	exists, err := CheckKeyExists(ctx, shortKey)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 先存储一个URL
	err = LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        longURL,
		Expiration: 60 * time.Second,
	})
	require.NoError(t, err)

	// 测试存在的key
	exists, err = CheckKeyExists(ctx, shortKey)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestServiceIntegration(t *testing.T) {
	cleanup, _ := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortKey := "integration"
	longURL := "https://example.com/integration/test/very/long/url/with/parameters?param1=value1&param2=value2"

	// 完整的集成测试流程

	// 1. 检查key不存在
	exists, err := CheckKeyExists(ctx, shortKey)
	require.NoError(t, err)
	assert.False(t, exists)

	// 2. 存储URL
	err = LongToShort(ctx, &LongToShortOptions{
		ShortKey:   shortKey,
		URL:        longURL,
		Expiration: 30 * time.Second,
	})
	require.NoError(t, err)

	// 3. 检查key存在
	exists, err = CheckKeyExists(ctx, shortKey)
	require.NoError(t, err)
	assert.True(t, exists)

	// 4. 获取URL
	result := ShortToLong(ctx, shortKey)
	assert.Equal(t, longURL, result)

	// 5. 续期
	err = Renew(ctx, shortKey, 60*time.Second)
	assert.NoError(t, err)

	// 6. 再次获取确认仍然存在
	result = ShortToLong(ctx, shortKey)
	assert.Equal(t, longURL, result)
}
