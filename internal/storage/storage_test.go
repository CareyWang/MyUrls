package storage

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isRedisAvailable 检查Redis服务器是否可用
func isRedisAvailable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestRedisDriver(t *testing.T) {
	// 跳过Redis测试如果设置了环境变量
	if os.Getenv("SKIP_REDIS_TESTS") != "" {
		t.Skip("Skipping Redis tests due to SKIP_REDIS_TESTS env var")
	}

	// 自动检测Redis是否可用
	redisAddr := "localhost:6379"
	if !isRedisAvailable(redisAddr) {
		t.Skipf("Skipping Redis tests - Redis server not available at %s", redisAddr)
	}

	driver, err := NewRedisDriver(redisAddr, "")
	if err != nil {
		t.Skipf("Skipping Redis tests - failed to connect: %v", err)
	}
	defer driver.Close()

	testStorageDriver(t, driver)
}

func TestSQLiteDriver(t *testing.T) {
	// 使用临时文件
	tmpFile := "./test_storage.db"
	defer os.Remove(tmpFile)

	driver, err := NewSQLiteDriver(tmpFile)
	require.NoError(t, err)
	defer driver.Close()

	testStorageDriver(t, driver)
}

// 通用的存储驱动测试
func testStorageDriver(t *testing.T, driver Driver) {
	ctx := context.Background()
	testKey := "test_key_" + time.Now().Format("20060102150405") // 使用时间戳确保key唯一
	testValue := "test_value"

	// 测试Ping
	err := driver.Ping(ctx)
	assert.NoError(t, err)

	// 测试Get不存在的key
	_, err = driver.Get(ctx, testKey)
	assert.Error(t, err)

	// 测试SetEx和Get
	err = driver.SetEx(ctx, testKey, testValue, 60*time.Second)
	assert.NoError(t, err)

	result, err := driver.Get(ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, result)

	// 测试Exists
	exists, err := driver.Exists(ctx, testKey)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试不存在的key
	exists, err = driver.Exists(ctx, "nonexistent_key")
	assert.NoError(t, err)
	assert.False(t, exists)

	// 测试TTL
	ttl, err := driver.TTL(ctx, testKey)
	assert.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= 60*time.Second)

	// 测试Expire
	err = driver.Expire(ctx, testKey, 30*time.Second)
	assert.NoError(t, err)

	// 验证TTL已更新
	newTTL, err := driver.TTL(ctx, testKey)
	assert.NoError(t, err)
	assert.True(t, newTTL <= 30*time.Second)

	// 测试过期功能 - 设置短过期时间（修复：改为2秒以兼容Redis最小过期时间要求）
	shortKey := "short_key"
	err = driver.SetEx(ctx, shortKey, "short_value", 2*time.Second)
	assert.NoError(t, err)

	// 立即验证key存在
	exists, err = driver.Exists(ctx, shortKey)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 等待过期（修复：等待3秒确保key完全过期）
	time.Sleep(3 * time.Second)

	// 验证key已过期
	_, err = driver.Get(ctx, shortKey)
	assert.Error(t, err)

	exists, err = driver.Exists(ctx, shortKey)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 清理测试数据
	// 注意：这里可能需要根据具体实现来清理
}

func TestStorageDriverInterface(t *testing.T) {
	// 测试Redis驱动实现了接口
	var _ Driver = (*RedisDriver)(nil)

	// 测试SQLite驱动实现了接口
	var _ Driver = (*SQLiteDriver)(nil)
}
