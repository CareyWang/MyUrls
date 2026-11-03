package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	sqliteBusyTimeoutMillis = 5000
	sqliteCleanupInterval   = 5 * time.Minute
)

// A mapping is valid while expires_at is NULL or greater than now.
const (
	createMappingsTableSQL = `CREATE TABLE IF NOT EXISTS url_mappings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL,
		expires_at INTEGER,
		created_at INTEGER
	)`
	createMappingsExpiryIndexSQL = `CREATE INDEX IF NOT EXISTS idx_url_mappings_expires_at ON url_mappings (expires_at)`
	claimMappingSQL              = `INSERT INTO url_mappings (key, value, expires_at, created_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at, created_at = excluded.created_at
		WHERE url_mappings.expires_at IS NOT NULL AND url_mappings.expires_at <= ?`
	resolveMappingSQL = `SELECT value FROM url_mappings WHERE key = ? AND (expires_at IS NULL OR expires_at > ?)`
	deleteExpiredSQL  = `DELETE FROM url_mappings WHERE expires_at IS NOT NULL AND expires_at <= ?`
)

// sqliteStore keeps mappings in a SQLite file for a single service instance.
type sqliteStore struct {
	db   *sql.DB
	now  func() time.Time
	stop chan struct{}
	done sync.WaitGroup
}

func newSQLiteStore(file string) (*sqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	// These pragmas apply per connection, so they belong in the DSN of the pool.
	dsn := url.URL{Scheme: "file", Path: file, RawQuery: fmt.Sprintf(
		"_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)", sqliteBusyTimeoutMillis,
	)}
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}
	for _, statement := range []string{createMappingsTableSQL, createMappingsExpiryIndexSQL} {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("prepare sqlite schema: %w", err)
		}
	}

	s := &sqliteStore{db: db, now: time.Now, stop: make(chan struct{})}
	s.done.Add(1)
	go s.cleanupExpired()
	return s, nil
}

// Claim is a single conditional upsert, so concurrent claims of a key have one winner.
func (s *sqliteStore) Claim(ctx context.Context, shortKey, longURL string, ttl time.Duration) (bool, error) {
	now := s.now()
	var expiresAt sql.NullInt64
	if ttl > 0 {
		expiresAt = sql.NullInt64{Int64: now.Add(ttl).Unix(), Valid: true}
	}
	result, err := s.db.ExecContext(ctx, claimMappingSQL, shortKey, longURL, expiresAt, now.Unix(), now.Unix())
	if err != nil {
		return false, err
	}
	written, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return written > 0, nil
}

func (s *sqliteStore) Resolve(ctx context.Context, shortKey string) (string, error) {
	var longURL string
	err := s.db.QueryRowContext(ctx, resolveMappingSQL, shortKey, s.now().Unix()).Scan(&longURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errShortURLNotFound
	}
	return longURL, err
}

func (s *sqliteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close stops the cleanup and releases the database.
func (s *sqliteStore) Close() error {
	close(s.stop)
	s.done.Wait()
	return s.db.Close()
}

// cleanupExpired only reclaims space, because queries already ignore expired mappings.
func (s *sqliteStore) cleanupExpired() {
	defer s.done.Done()
	ticker := time.NewTicker(sqliteCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			if err := s.deleteExpired(); err != nil {
				logger.Warn("failed to delete expired mappings: ", err.Error())
			}
		}
	}
}

func (s *sqliteStore) deleteExpired() error {
	_, err := s.db.Exec(deleteExpiredSQL, s.now().Unix())
	return err
}
