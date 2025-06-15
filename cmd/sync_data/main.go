package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/CareyWang/MyUrls/internal/logger"
	"github.com/CareyWang/MyUrls/internal/storage"
	"github.com/redis/go-redis/v9"
)

var (
	helpFlag      bool
	redisAddr     string
	redisPassword string
	sqliteFile    string
	batchSize     int
)

func init() {
	flag.BoolVar(&helpFlag, "h", false, "显示帮助信息")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis 地址")
	flag.StringVar(&redisPassword, "redis-password", "", "Redis 密码")
	flag.StringVar(&sqliteFile, "sqlite-file", "./data/myurls.db", "SQLite 文件路径")
	flag.IntVar(&batchSize, "batch-size", 100, "批量处理大小")
}

func main() {
	flag.Parse()
	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// 从环境变量中读取配置
	parseEnvirons()

	logger.Init()
	logger.Logger.Info("sync_data app 启动中...")

	// 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           0,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		MaxRetries:   1,
	})
	defer redisClient.Close()

	// 测试 Redis 连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Logger.Fatalf("Redis 连接失败: %v", err)
	}
	logger.Logger.Info("Redis 连接成功")

	// 初始化 SQLite 存储
	sqliteDriver, err := storage.NewSQLiteDriver(sqliteFile)
	if err != nil {
		logger.Logger.Fatalf("SQLite 初始化失败: %v", err)
	}
	defer sqliteDriver.Close()

	// 测试 SQLite 连接
	if err := sqliteDriver.Ping(ctx); err != nil {
		logger.Logger.Fatalf("SQLite 连接失败: %v", err)
	}
	logger.Logger.Info("SQLite 连接成功")

	// 执行一次性同步
	syncer := &DataSyncer{
		redisClient:  redisClient,
		sqliteDriver: sqliteDriver,
		batchSize:    batchSize,
	}

	logger.Logger.Infof("开始一次性数据同步，批量大小: %d", batchSize)
	if err := syncer.syncData(ctx); err != nil {
		logger.Logger.Fatalf("同步失败: %v", err)
	}
	logger.Logger.Info("同步完成，程序退出")
}

func parseEnvirons() {
	if addr := os.Getenv("SYNC_REDIS_ADDR"); addr != "" {
		redisAddr = addr
	}
	if password := os.Getenv("SYNC_REDIS_PASSWORD"); password != "" {
		redisPassword = password
	}
	if file := os.Getenv("SYNC_SQLITE_FILE"); file != "" {
		sqliteFile = file
	}
}

type DataSyncer struct {
	redisClient  *redis.Client
	sqliteDriver storage.Driver
	batchSize    int
}

func (s *DataSyncer) syncData(ctx context.Context) error {
	logger.Logger.Info("开始同步数据...")

	var cursor uint64
	syncedCount := 0
	errorCount := 0
	totalKeys := 0
	batchCount := 0
	startTime := time.Now()

	for {
		keys, nextCursor, err := s.redisClient.Scan(ctx, cursor, "*", int64(s.batchSize)).Result()
		if err != nil {
			return fmt.Errorf("扫描 Redis keys 失败: %w", err)
		}

		totalKeys += len(keys)
		batchCount++

		// 立即处理当前批次的keys
		if len(keys) > 0 {
			logger.Logger.Infof("处理第 %d 批次，包含 %d 个 keys", batchCount, len(keys))
			if err := s.syncBatch(ctx, keys); err != nil {
				logger.Logger.Errorf("批量同步失败: %v", err)
				errorCount++
			} else {
				syncedCount += len(keys)
			}

			// 显示进度信息
			elapsed := time.Since(startTime)
			logger.Logger.Infof("进度更新: 已处理 %d 个 keys，成功 %d 个，用时 %v", totalKeys, syncedCount, elapsed.Truncate(time.Second))
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	logger.Logger.Infof("找到 %d 个 Redis keys", totalKeys)

	if totalKeys == 0 {
		logger.Logger.Info("没有找到需要同步的数据")
		return nil
	}

	totalTime := time.Since(startTime)
	successRate := float64(syncedCount) / float64(totalKeys) * 100
	logger.Logger.Infof("同步完成: 总计 %d 个 keys，成功 %d 个 (%.1f%%)，失败 %d 个批次，总用时 %v",
		totalKeys, syncedCount, successRate, errorCount, totalTime.Truncate(time.Second))

	return nil
}

func (s *DataSyncer) syncBatch(ctx context.Context, keys []string) error {
	for _, key := range keys {
		// 获取 Redis 中的值
		value, err := s.redisClient.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				// key 不存在，跳过
				continue
			}
			return fmt.Errorf("获取 Redis key %s 失败: %w", key, err)
		}

		// 获取 TTL
		ttl, err := s.redisClient.TTL(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("获取 Redis key %s TTL 失败: %w", key, err)
		}

		// 同步到 SQLite
		if ttl == -1 {
			// 永不过期的 key，设置一个很长的过期时间
			err = s.sqliteDriver.SetEx(ctx, key, value, 10*365*24*time.Hour)
		} else if ttl > 0 {
			// 有过期时间的 key
			err = s.sqliteDriver.SetEx(ctx, key, value, ttl)
		} else {
			// TTL 为 -2 表示 key 不存在，跳过
			continue
		}

		if err != nil {
			return fmt.Errorf("同步 key %s 到 SQLite 失败: %w", key, err)
		}
	}

	return nil
}
