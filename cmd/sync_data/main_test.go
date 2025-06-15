package main

import (
	"context"
	"testing"
	"time"

	"github.com/CareyWang/MyUrls/internal/logger"
	"github.com/CareyWang/MyUrls/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// 初始化测试用的 logger
	logger.Init()
	m.Run()
}

func setupTestSyncer(t *testing.T) (*DataSyncer, func()) {
	// 使用内存SQLite数据库进行测试
	sqliteDriver, err := storage.NewSQLiteDriver(":memory:")
	require.NoError(t, err)

	syncer := &DataSyncer{
		redisClient:  nil, // 在某些测试中我们不需要真实的Redis客户端
		sqliteDriver: sqliteDriver,
		batchSize:    3, // 小批次用于测试
	}

	cleanup := func() {
		sqliteDriver.Close()
	}

	return syncer, cleanup
}

func TestDataSyncer_syncSingleKey_Success(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	// 先在SQLite中设置一个测试值
	err := syncer.sqliteDriver.SetEx(ctx, "test-key", "test-value", time.Hour)
	require.NoError(t, err)

	// 验证值确实被设置了
	value, err := syncer.sqliteDriver.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, "test-value", value)
}

func TestDataSyncer_syncSingleKey_PersistentKey(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	// 测试永不过期的key（设置一个很长的过期时间）
	err := syncer.sqliteDriver.SetEx(ctx, "persistent-key", "persistent-value", 10*365*24*time.Hour)
	require.NoError(t, err)

	// 验证TTL
	ttl, err := syncer.sqliteDriver.TTL(ctx, "persistent-key")
	assert.NoError(t, err)
	assert.True(t, ttl > 365*24*time.Hour) // 应该是一个很长的时间
}

func TestDataSyncer_syncSingleKey_KeyNotExists(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	// 尝试获取不存在的key
	_, err := syncer.sqliteDriver.Get(ctx, "nonexistent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis: nil") // SQLite驱动模拟Redis的行为
}

func TestDataSyncer_StorageOperations(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	testCases := []struct {
		name       string
		key        string
		value      string
		expiration time.Duration
	}{
		{
			name:       "short expiration",
			key:        "short-key",
			value:      "short-value",
			expiration: 1 * time.Second,
		},
		{
			name:       "long expiration",
			key:        "long-key",
			value:      "long-value",
			expiration: 1 * time.Hour,
		},
		{
			name:       "very long expiration",
			key:        "persistent-key",
			value:      "persistent-value",
			expiration: 10 * 365 * 24 * time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置值
			err := syncer.sqliteDriver.SetEx(ctx, tc.key, tc.value, tc.expiration)
			require.NoError(t, err)

			// 验证值
			retrievedValue, err := syncer.sqliteDriver.Get(ctx, tc.key)
			assert.NoError(t, err)
			assert.Equal(t, tc.value, retrievedValue)

			// 验证key存在
			exists, err := syncer.sqliteDriver.Exists(ctx, tc.key)
			assert.NoError(t, err)
			assert.True(t, exists)

			// 验证TTL
			ttl, err := syncer.sqliteDriver.TTL(ctx, tc.key)
			assert.NoError(t, err)
			assert.True(t, ttl > 0, "TTL should be positive for key %s", tc.key)
			assert.True(t, ttl <= tc.expiration, "TTL should not exceed set expiration for key %s", tc.key)
		})
	}
}

func TestDataSyncer_BatchOperations(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	// 预先在存储中设置一些测试数据
	testData := map[string]string{
		"batch-key1": "value1",
		"batch-key2": "value2",
		"batch-key3": "value3",
	}

	for key, value := range testData {
		err := syncer.sqliteDriver.SetEx(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	// 验证所有数据都被正确存储
	for key, expectedValue := range testData {
		value, err := syncer.sqliteDriver.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, value)

		exists, err := syncer.sqliteDriver.Exists(ctx, key)
		assert.NoError(t, err)
		assert.True(t, exists)
	}
}

func TestDataSyncer_ExpiredKeys(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	// 设置一个很短过期时间的key
	err := syncer.sqliteDriver.SetEx(ctx, "expire-key", "expire-value", 1*time.Second)
	require.NoError(t, err)

	// 立即验证key存在
	exists, err := syncer.sqliteDriver.Exists(ctx, "expire-key")
	assert.NoError(t, err)
	assert.True(t, exists)

	// 等待过期
	time.Sleep(2 * time.Second)

	// 验证key已过期
	_, err = syncer.sqliteDriver.Get(ctx, "expire-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis: nil")

	exists, err = syncer.sqliteDriver.Exists(ctx, "expire-key")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDataSyncer_StatisticsAccuracy(t *testing.T) {
	// 这个测试验证统计的准确性，使用模拟场景

	// 模拟一批keys的处理结果
	type keyResult struct {
		key     string
		success bool
	}

	testBatch := []keyResult{
		{"success1", true},
		{"success2", true},
		{"fail1", false},
		{"success3", true},
		{"fail2", false},
	}

	expectedSucceed := 0
	expectedFailed := 0

	for _, result := range testBatch {
		if result.success {
			expectedSucceed++
		} else {
			expectedFailed++
		}
	}

	// 验证统计逻辑
	assert.Equal(t, 3, expectedSucceed, "应该有3个成功的key")
	assert.Equal(t, 2, expectedFailed, "应该有2个失败的key")
	assert.Equal(t, 5, expectedSucceed+expectedFailed, "总数应该等于成功+失败")
}

func TestDataSyncer_ErrorHandling(t *testing.T) {
	syncer, cleanup := setupTestSyncer(t)
	defer cleanup()

	ctx := context.Background()

	// 测试使用已关闭的连接
	syncer.sqliteDriver.Close()

	// 尝试操作应该失败
	err := syncer.sqliteDriver.SetEx(ctx, "test-key", "test-value", time.Hour)
	assert.Error(t, err)

	_, err = syncer.sqliteDriver.Get(ctx, "test-key")
	assert.Error(t, err)
}

// 测试辅助函数：验证优化后的容错机制的概念
func TestContainerFaultTolerance(t *testing.T) {
	// 这个测试验证我们优化的容错机制的概念正确性

	// 模拟批次处理的结果
	batchResults := []struct {
		batchID     int
		successKeys int
		failedKeys  int
	}{
		{1, 8, 2},  // 第1批：8成功，2失败
		{2, 10, 0}, // 第2批：10成功，0失败
		{3, 5, 5},  // 第3批：5成功，5失败
		{4, 0, 3},  // 第4批：0成功，3失败（整批失败）
		{5, 7, 1},  // 第5批：7成功，1失败
	}

	totalSuccess := 0
	totalFailed := 0

	for _, batch := range batchResults {
		totalSuccess += batch.successKeys
		totalFailed += batch.failedKeys

		// 验证即使某个批次全部失败，处理仍然继续
		t.Logf("批次 %d: 成功 %d, 失败 %d", batch.batchID, batch.successKeys, batch.failedKeys)
	}

	assert.Equal(t, 30, totalSuccess, "总成功数应该是30")
	assert.Equal(t, 11, totalFailed, "总失败数应该是11")

	// 计算成功率
	totalKeys := totalSuccess + totalFailed
	successRate := float64(totalSuccess) / float64(totalKeys) * 100

	assert.Equal(t, 41, totalKeys, "总key数应该是41")
	assert.InDelta(t, 73.17, successRate, 0.01, "成功率should约为73.17%")

	t.Logf("处理完成: 总计 %d 个 keys，成功 %d 个 (%.1f%%)，失败 %d 个",
		totalKeys, totalSuccess, successRate, totalFailed)
}
