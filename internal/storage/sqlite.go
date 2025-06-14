package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDriver struct {
	db *sql.DB
}

func NewSQLiteDriver(filePath string) (*SQLiteDriver, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, err
	}

	driver := &SQLiteDriver{db: db}

	// 初始化表结构
	if err := driver.initTables(); err != nil {
		return nil, err
	}

	// 启动清理过期数据的goroutine
	go driver.cleanupExpiredKeys()

	return driver, nil
}

func (s *SQLiteDriver) initTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS url_mappings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		expires_at INTEGER,
		created_at INTEGER DEFAULT (strftime('%s', 'now'))
	);
	
	CREATE INDEX IF NOT EXISTS idx_expires_at ON url_mappings(expires_at);
	`

	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteDriver) Get(ctx context.Context, key string) (string, error) {
	var value string
	var expiresAt *int64

	query := `SELECT value, expires_at FROM url_mappings WHERE key = ?`
	err := s.db.QueryRowContext(ctx, query, key).Scan(&value, &expiresAt)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("redis: nil") // 模拟Redis的行为
	}
	if err != nil {
		return "", err
	}

	// 检查是否过期
	if expiresAt != nil && time.Now().Unix() > *expiresAt {
		// 删除过期的key
		s.db.ExecContext(ctx, `DELETE FROM url_mappings WHERE key = ?`, key)
		return "", fmt.Errorf("redis: nil") // 模拟Redis的行为
	}

	return value, nil
}

func (s *SQLiteDriver) SetEx(ctx context.Context, key string, value string, expiration time.Duration) error {
	expiresAt := time.Now().Add(expiration).Unix()

	query := `INSERT OR REPLACE INTO url_mappings (key, value, expires_at) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, key, value, expiresAt)
	return err
}

func (s *SQLiteDriver) Exists(ctx context.Context, key string) (bool, error) {
	var expiresAt *int64

	query := `SELECT expires_at FROM url_mappings WHERE key = ?`
	err := s.db.QueryRowContext(ctx, query, key).Scan(&expiresAt)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 检查是否过期
	if expiresAt != nil && time.Now().Unix() > *expiresAt {
		// 删除过期的key
		s.db.ExecContext(ctx, `DELETE FROM url_mappings WHERE key = ?`, key)
		return false, nil
	}

	return true, nil
}

func (s *SQLiteDriver) TTL(ctx context.Context, key string) (time.Duration, error) {
	var expiresAt *int64

	query := `SELECT expires_at FROM url_mappings WHERE key = ?`
	err := s.db.QueryRowContext(ctx, query, key).Scan(&expiresAt)

	if err == sql.ErrNoRows {
		return -2 * time.Second, nil // key不存在
	}
	if err != nil {
		return 0, err
	}

	if expiresAt == nil {
		return -1 * time.Second, nil // 永不过期
	}

	now := time.Now().Unix()
	if now > *expiresAt {
		return -2 * time.Second, nil // 已过期
	}

	return time.Duration(*expiresAt-now) * time.Second, nil
}

func (s *SQLiteDriver) Expire(ctx context.Context, key string, expiration time.Duration) error {
	expiresAt := time.Now().Add(expiration).Unix()

	query := `UPDATE url_mappings SET expires_at = ? WHERE key = ?`
	result, err := s.db.ExecContext(ctx, query, expiresAt, key)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("key not found")
	}

	return nil
}

func (s *SQLiteDriver) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLiteDriver) Close() error {
	return s.db.Close()
}

// 清理过期数据的后台任务
func (s *SQLiteDriver) cleanupExpiredKeys() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		s.db.Exec(`DELETE FROM url_mappings WHERE expires_at IS NOT NULL AND expires_at < ?`, now)
	}
}
