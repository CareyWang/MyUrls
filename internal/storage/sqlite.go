package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 完全静默的logger，不输出任何日志
type SilentLogger struct{}

func (l SilentLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l SilentLogger) Info(context.Context, string, ...interface{}) {}

func (l SilentLogger) Warn(context.Context, string, ...interface{}) {}

func (l SilentLogger) Error(context.Context, string, ...interface{}) {}

func (l SilentLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
}

type URLMapping struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Key       string `gorm:"index;unique" json:"key"`
	Value     string `gorm:"not null" json:"value"`
	ExpiresAt *int64 `gorm:"index" json:"expires_at"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at"`
}

func (URLMapping) TableName() string {
	return "url_mappings"
}

type SQLiteDriver struct {
	db *gorm.DB
}

func NewSQLiteDriver(filePath string) (*SQLiteDriver, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 配置完全静默的自定义logger
	db, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{
		Logger: SilentLogger{},
	})
	if err != nil {
		return nil, err
	}

	driver := &SQLiteDriver{db: db}

	// 自动迁移表结构
	if err := driver.db.AutoMigrate(&URLMapping{}); err != nil {
		return nil, err
	}

	// 启动清理过期数据的goroutine
	go driver.cleanupExpiredKeys()

	return driver, nil
}

func (s *SQLiteDriver) Get(ctx context.Context, key string) (string, error) {
	var mapping URLMapping
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&mapping).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("redis: nil") // 模拟Redis的行为
	}
	if err != nil {
		return "", err
	}

	// 检查是否过期
	if mapping.ExpiresAt != nil && time.Now().Unix() > *mapping.ExpiresAt {
		// 删除过期的key
		s.db.WithContext(ctx).Delete(&URLMapping{}, "key = ?", key)
		return "", fmt.Errorf("redis: nil") // 模拟Redis的行为
	}

	return mapping.Value, nil
}

func (s *SQLiteDriver) SetEx(ctx context.Context, key string, value string, expiration time.Duration) error {
	expiresAt := time.Now().Add(expiration).Unix()

	// 使用 ON CONFLICT 子句来处理 upsert 操作
	result := s.db.WithContext(ctx).
		Exec("INSERT INTO url_mappings (key, value, expires_at, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at",
			key, value, expiresAt, time.Now().Unix())

	return result.Error
}

func (s *SQLiteDriver) Exists(ctx context.Context, key string) (bool, error) {
	var mapping URLMapping
	err := s.db.WithContext(ctx).Select("expires_at").Where("key = ?", key).First(&mapping).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 检查是否过期
	if mapping.ExpiresAt != nil && time.Now().Unix() > *mapping.ExpiresAt {
		// 删除过期的key
		s.db.WithContext(ctx).Delete(&URLMapping{}, "key = ?", key)
		return false, nil
	}

	return true, nil
}

func (s *SQLiteDriver) TTL(ctx context.Context, key string) (time.Duration, error) {
	var mapping URLMapping
	err := s.db.WithContext(ctx).Select("expires_at").Where("key = ?", key).First(&mapping).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return -2 * time.Second, nil // key不存在
	}
	if err != nil {
		return 0, err
	}

	if mapping.ExpiresAt == nil {
		return -1 * time.Second, nil // 永不过期
	}

	now := time.Now().Unix()
	if now > *mapping.ExpiresAt {
		return -2 * time.Second, nil // 已过期
	}

	return time.Duration(*mapping.ExpiresAt-now) * time.Second, nil
}

func (s *SQLiteDriver) Expire(ctx context.Context, key string, expiration time.Duration) error {
	expiresAt := time.Now().Add(expiration).Unix()

	result := s.db.WithContext(ctx).Model(&URLMapping{}).Where("key = ?", key).Update("expires_at", expiresAt)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("key not found")
	}

	return nil
}

func (s *SQLiteDriver) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *SQLiteDriver) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// 清理过期数据的后台任务
func (s *SQLiteDriver) cleanupExpiredKeys() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		s.db.Delete(&URLMapping{}, "expires_at IS NOT NULL AND expires_at < ?", now)
	}
}
