package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestSync(t *testing.T) (*miniredis.Miniredis, *redisStore, *sqliteStore, *testClock) {
	t.Helper()
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	server := miniredis.RunT(t)
	source := newRedisStore(server.Addr(), "")
	t.Cleanup(func() { source.Close() })
	target, clock := newTestSQLiteStore(t)
	return server, source, target, clock
}

func TestSyncMappingsKeepsLifetimes(t *testing.T) {
	server, source, target, clock := newTestSync(t)
	require.NoError(t, server.Set("expiring", "https://example.com/expiring"))
	server.SetTTL("expiring", time.Hour)
	require.NoError(t, server.Set("forever", "https://example.com/forever"))

	result, err := syncMappings(context.Background(), source, target, 10)
	require.NoError(t, err)
	require.Equal(t, syncResult{synced: 2}, result)
	requireStored(t, target, "expiring", "https://example.com/expiring")
	requireStored(t, target, "forever", "https://example.com/forever")

	_, life, err := target.Resolve(context.Background(), "expiring")
	require.NoError(t, err)
	require.Equal(t, remainingLifetime(time.Hour), life)
	_, life, err = target.Resolve(context.Background(), "forever")
	require.NoError(t, err)
	require.Equal(t, permanentLifetime, life, "a permanent key must stay permanent")

	clock.advance(time.Hour)
	_, _, err = target.Resolve(context.Background(), "expiring")
	require.ErrorIs(t, err, errShortURLNotFound)
	requireStored(t, target, "forever", "https://example.com/forever")
}

func TestSyncMappingsSkipsKeysAboutToExpire(t *testing.T) {
	server, source, target, _ := newTestSync(t)
	require.NoError(t, server.Set("expiring", "https://example.com/expiring"))
	server.SetTTL("expiring", 500*time.Microsecond) // PTTL reports 0.

	result, err := syncMappings(context.Background(), source, target, 10)
	require.NoError(t, err)
	require.Equal(t, syncResult{skipped: 1}, result)
	_, _, err = target.Resolve(context.Background(), "expiring")
	require.ErrorIs(t, err, errShortURLNotFound, "an expiring key must not become permanent")
}

func TestSyncMappingsDoesNotApplyCreationRules(t *testing.T) {
	server, source, target, _ := newTestSync(t)
	require.NoError(t, server.Set("short", "javascript:alert(1)"))
	require.NoError(t, server.Set("logo.png", "https://example.com/legacy"))

	result, err := syncMappings(context.Background(), source, target, 10)
	require.NoError(t, err)
	require.Equal(t, syncResult{synced: 2}, result)
	requireStored(t, target, "short", "javascript:alert(1)")
	requireStored(t, target, "logo.png", "https://example.com/legacy")
}

func TestSyncMappingsDoesNotOverwriteAndIsRepeatable(t *testing.T) {
	server, source, target, _ := newTestSync(t)
	ctx := context.Background()
	require.NoError(t, server.Set("conflict", "https://example.com/from-redis"))
	require.NoError(t, server.Set("new", "https://example.com/new"))
	_, err := target.Claim(ctx, "conflict", "https://example.com/from-sqlite", remainingLifetime(time.Hour))
	require.NoError(t, err)

	result, err := syncMappings(ctx, source, target, 10)
	require.NoError(t, err)
	require.Equal(t, syncResult{synced: 1, skipped: 1}, result)
	requireStored(t, target, "conflict", "https://example.com/from-sqlite")
	requireStored(t, target, "new", "https://example.com/new")

	result, err = syncMappings(ctx, source, target, 10)
	require.NoError(t, err)
	require.Equal(t, syncResult{skipped: 2}, result)
}

func TestSyncMappingsCountsNonStringKeysAsFailed(t *testing.T) {
	server, source, target, _ := newTestSync(t)
	require.NoError(t, server.Set("good", "https://example.com/good"))
	_, err := server.Lpush("not-a-string", "value")
	require.NoError(t, err)

	result, err := syncMappings(context.Background(), source, target, 10)
	require.NoError(t, err)
	require.Equal(t, syncResult{synced: 1, failed: 1}, result)
	requireStored(t, target, "good", "https://example.com/good")
}

func TestSyncMappingsCoversEveryBatch(t *testing.T) {
	server, source, target, _ := newTestSync(t)
	const keys = 25
	for i := range keys {
		require.NoError(t, server.Set(fmt.Sprintf("key%d", i), "https://example.com"))
	}

	result, err := syncMappings(context.Background(), source, target, 3)
	require.NoError(t, err)
	require.Equal(t, syncResult{synced: keys}, result)
}

func TestSyncMappingsReportsScanFailure(t *testing.T) {
	server, source, target, _ := newTestSync(t)
	server.SetError("ERR storage unavailable")

	_, err := syncMappings(context.Background(), source, target, 10)
	require.ErrorContains(t, err, "scan redis keys")
}

// prepareSyncCommand restores the globals that the command sets and returns a database path.
func prepareSyncCommand(t *testing.T) string {
	t.Helper()
	prepareRun(t)
	return filepath.Join(t.TempDir(), "myurls.db")
}

func TestRunSyncRedisToSQLite(t *testing.T) {
	file := prepareSyncCommand(t)
	server := miniredis.RunT(t)
	require.NoError(t, server.Set("key", "https://example.com"))

	require.NoError(t, runSyncRedisToSQLite([]string{"-conn", server.Addr(), "-sqlite-file", file}))

	db, err := sql.Open("sqlite", file)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	var longURL string
	require.NoError(t, db.QueryRow("SELECT value FROM url_mappings WHERE key = 'key'").Scan(&longURL))
	require.Equal(t, "https://example.com", longURL)
}

func TestRunSyncRedisToSQLiteFailsOnPartialSync(t *testing.T) {
	file := prepareSyncCommand(t)
	server := miniredis.RunT(t)
	require.NoError(t, server.Set("good", "https://example.com"))
	_, err := server.Lpush("not-a-string", "value")
	require.NoError(t, err)

	err = runSyncRedisToSQLite([]string{"-conn", server.Addr(), "-sqlite-file", file})
	require.EqualError(t, err, "failed to sync 1 keys")
}

func TestRunSyncRedisToSQLiteFailsWithoutRedis(t *testing.T) {
	file := prepareSyncCommand(t)
	server := miniredis.RunT(t)
	addr := server.Addr()
	server.Close()

	err := runSyncRedisToSQLite([]string{"-conn", addr, "-sqlite-file", file})
	require.ErrorContains(t, err, "redis ping failed")
	require.NoFileExists(t, file)
}
