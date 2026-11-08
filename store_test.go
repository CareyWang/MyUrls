package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

// testClock is a controllable time source for sqliteStore.
type testClock struct {
	mu      sync.Mutex
	current time.Time
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
}

func newTestSQLiteStore(t testing.TB) (*sqliteStore, *testClock) {
	t.Helper()
	store, err := newSQLiteStore(filepath.Join(t.TempDir(), "myurls.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	clock := &testClock{current: time.Unix(1_800_000_000, 0)}
	store.now = clock.now
	return store, clock
}

// forEachStore runs a behavior that every storage backend must share.
// advance moves the backend's clock forward.
func forEachStore(t *testing.T, test func(t *testing.T, store mappingStore, advance func(time.Duration))) {
	t.Run("redis", func(t *testing.T) {
		server := miniredis.RunT(t)
		store := newRedisStore(server.Addr(), "")
		t.Cleanup(func() { store.Close() })
		test(t, store, server.FastForward)
	})
	t.Run("sqlite", func(t *testing.T) {
		store, clock := newTestSQLiteStore(t)
		test(t, store, clock.advance)
	})
}

func requireStored(t *testing.T, store mappingStore, shortKey, wantLongURL string) {
	t.Helper()
	longURL, _, err := store.Resolve(context.Background(), shortKey)
	require.NoError(t, err)
	require.Equal(t, wantLongURL, longURL)
}

func TestStoreResolvesMissingKeyAsNotFound(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		longURL, _, err := store.Resolve(context.Background(), "missing")
		require.ErrorIs(t, err, errShortURLNotFound)
		require.Empty(t, longURL)
	})
}

func TestStoreClaimDoesNotOverwriteValidMapping(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		claimed, err := store.Claim(ctx, "key", "https://example.com/original", remainingLifetime(time.Hour))
		require.NoError(t, err)
		require.True(t, claimed)

		advance(59 * time.Minute)
		claimed, err = store.Claim(ctx, "key", "https://example.com/replacement", remainingLifetime(time.Hour))
		require.NoError(t, err)
		require.False(t, claimed)
		requireStored(t, store, "key", "https://example.com/original")

		// The failed claim must not have extended the original expiration.
		advance(time.Minute)
		_, _, err = store.Resolve(ctx, "key")
		require.ErrorIs(t, err, errShortURLNotFound)
	})
}

func TestStoreClaimReplacesExpiredMapping(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		_, err := store.Claim(ctx, "key", "https://example.com/original", remainingLifetime(time.Hour))
		require.NoError(t, err)

		advance(time.Hour)
		claimed, err := store.Claim(ctx, "key", "https://example.com/replacement", remainingLifetime(time.Hour))
		require.NoError(t, err)
		require.True(t, claimed)
		requireStored(t, store, "key", "https://example.com/replacement")
	})
}

func TestStorePermanentLifetimeNeverExpires(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		claimed, err := store.Claim(ctx, "key", "https://example.com/original", permanentLifetime)
		require.NoError(t, err)
		require.True(t, claimed)

		advance(100 * 365 * 24 * time.Hour)
		requireStored(t, store, "key", "https://example.com/original")
		claimed, err = store.Claim(ctx, "key", "https://example.com/replacement", remainingLifetime(time.Hour))
		require.NoError(t, err)
		require.False(t, claimed)
	})
}

func TestStoreResolveReportsRemainingLifetime(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		_, err := store.Claim(ctx, "expiring", "https://example.com", remainingLifetime(time.Hour))
		require.NoError(t, err)
		advance(20 * time.Minute)
		_, life, err := store.Resolve(ctx, "expiring")
		require.NoError(t, err)
		require.Equal(t, remainingLifetime(40*time.Minute), life)

		_, err = store.Claim(ctx, "forever", "https://example.com", permanentLifetime)
		require.NoError(t, err)
		_, life, err = store.Resolve(ctx, "forever")
		require.NoError(t, err)
		require.Equal(t, permanentLifetime, life)

		advance(40 * time.Minute)
		_, _, err = store.Resolve(ctx, "expiring")
		require.ErrorIs(t, err, errShortURLNotFound)
	})
}

func TestStoreClaimRejectsInvalidLifetime(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		ctx := context.Background()
		for name, life := range map[string]lifetime{
			"zero value":         {},
			"zero remaining":     remainingLifetime(0),
			"negative remaining": remainingLifetime(-time.Second),
		} {
			claimed, err := store.Claim(ctx, "key", "https://example.com", life)
			require.ErrorIs(t, err, errInvalidLifetime, name)
			require.False(t, claimed, name)
		}
		_, _, err := store.Resolve(ctx, "key")
		require.ErrorIs(t, err, errShortURLNotFound, "a rejected claim must store nothing")
	})
}

func TestStoreClaimSubSecondLifetime(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		claimed, err := store.Claim(ctx, "key", "https://example.com", remainingLifetime(500*time.Millisecond))
		require.NoError(t, err)
		require.True(t, claimed)
		requireStored(t, store, "key", "https://example.com")

		// SQLite keeps whole seconds, so it may outlive Redis by up to a second.
		advance(time.Second)
		_, _, err = store.Resolve(ctx, "key")
		require.ErrorIs(t, err, errShortURLNotFound)
	})
}

func TestSQLiteStoreRoundsExpiryUpToWholeSeconds(t *testing.T) {
	store, clock := newTestSQLiteStore(t)
	clock.advance(300 * time.Millisecond)

	// Rounding down would make this mapping expire before it was written.
	claimed, err := store.Claim(context.Background(), "key", "https://example.com", remainingLifetime(500*time.Millisecond))
	require.NoError(t, err)
	require.True(t, claimed)
	_, life, err := store.Resolve(context.Background(), "key")
	require.NoError(t, err)
	require.Equal(t, remainingLifetime(700*time.Millisecond), life)
}

func TestRedisStoreResolvesKeyUnderOneMillisecondAsNotFound(t *testing.T) {
	server := miniredis.RunT(t)
	store := newRedisStore(server.Addr(), "")
	t.Cleanup(func() { store.Close() })
	require.NoError(t, server.Set("key", "https://example.com"))
	server.SetTTL("key", 500*time.Microsecond) // PTTL reports 0.

	_, _, err := store.Resolve(context.Background(), "key")
	require.ErrorIs(t, err, errShortURLNotFound)
}

func TestStoreConcurrentClaimsHaveOneWinner(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		const requests = 32
		errs := make([]error, requests)
		won := make([]bool, requests)
		var wg sync.WaitGroup
		for i := range requests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				won[i], errs[i] = store.Claim(context.Background(), "contended", fmt.Sprintf("https://example.com/%d", i), remainingLifetime(time.Hour))
			}()
		}
		wg.Wait()

		winner := -1
		for i := range requests {
			require.NoError(t, errs[i])
			if won[i] {
				require.Equal(t, -1, winner, "only one claim may win")
				winner = i
			}
		}
		require.NotEqual(t, -1, winner)
		requireStored(t, store, "contended", fmt.Sprintf("https://example.com/%d", winner))
	})
}

func TestStoreReportsFailureAsErrorNotAbsence(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := store.Resolve(ctx, "key")
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, errShortURLNotFound)

		claimed, err := store.Claim(ctx, "key", "https://example.com", remainingLifetime(time.Hour))
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, claimed)
	})
}

func TestMappingsRetryCollisionsOnEveryStore(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		ctx := context.Background()
		_, err := store.Claim(ctx, "taken", "https://example.com/original", remainingLifetime(time.Hour))
		require.NoError(t, err)

		mappings := &shortURLMappings{store: store}
		candidates := []string{"taken", "fresh"}
		mappings.generateKey = func() (string, error) {
			key := candidates[0]
			candidates = candidates[1:]
			return key, nil
		}

		key, err := mappings.Create(ctx, "https://example.com/new", "")
		require.NoError(t, err)
		require.Equal(t, "fresh", key)
		requireStored(t, store, "taken", "https://example.com/original")
		requireStored(t, store, "fresh", "https://example.com/new")
	})
}

func TestSQLiteStoreKeepsMappingsAcrossReopen(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "dir", "myurls.db")
	store, err := newSQLiteStore(file)
	require.NoError(t, err)
	claimed, err := store.Claim(context.Background(), "key", "https://example.com", remainingLifetime(time.Hour))
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, store.Close())

	reopened, err := newSQLiteStore(file)
	require.NoError(t, err)
	t.Cleanup(func() { reopened.Close() })
	requireStored(t, reopened, "key", "https://example.com")
}

func TestSQLiteStoreOpensFilePaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		file func(dir string) string
	}{
		{"default relative path", func(string) string { return sqliteFile }},
		{"bare file name", func(string) string { return "myurls.db" }},
		{"absolute path", func(dir string) string { return filepath.Join(dir, "myurls.db") }},
		{"reserved URI characters", func(dir string) string { return filepath.Join(dir, "with space", "a?b#c%20.db") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			file := tc.file(dir)

			store, err := newSQLiteStore(file)
			require.NoError(t, err)
			t.Cleanup(func() { store.Close() })
			_, err = store.Claim(context.Background(), "key", "https://example.com", permanentLifetime)
			require.NoError(t, err)
			require.FileExists(t, file, "the database must be at the exact configured path")
		})
	}
}

func TestSQLiteStoreConfiguresEveryConnection(t *testing.T) {
	store, _ := newTestSQLiteStore(t)
	ctx := context.Background()

	// Holding connections open forces the pool to create more of them.
	for range 3 {
		conn, err := store.db.Conn(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { conn.Close() })

		var busyTimeout int
		require.NoError(t, conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
		require.Equal(t, sqliteBusyTimeoutMillis, busyTimeout)
		var journalMode string
		require.NoError(t, conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
		require.Equal(t, "wal", journalMode)
	}
}

func TestSQLiteStoreDeleteExpiredKeepsValidMappings(t *testing.T) {
	store, clock := newTestSQLiteStore(t)
	ctx := context.Background()
	for shortKey, life := range map[string]lifetime{"expired": remainingLifetime(time.Hour), "valid": remainingLifetime(2 * time.Hour), "forever": permanentLifetime} {
		_, err := store.Claim(ctx, shortKey, "https://example.com/"+shortKey, life)
		require.NoError(t, err)
	}

	clock.advance(time.Hour)
	// Recreating an expired key must survive the cleanup.
	_, err := store.Claim(ctx, "recreated", "https://example.com/old", remainingLifetime(time.Minute))
	require.NoError(t, err)
	clock.advance(time.Minute)
	claimed, err := store.Claim(ctx, "recreated", "https://example.com/new", remainingLifetime(time.Hour))
	require.NoError(t, err)
	require.True(t, claimed)

	require.NoError(t, store.deleteExpired())

	var remaining int
	require.NoError(t, store.db.QueryRow("SELECT COUNT(*) FROM url_mappings").Scan(&remaining))
	require.Equal(t, 3, remaining)
	requireStored(t, store, "valid", "https://example.com/valid")
	requireStored(t, store, "forever", "https://example.com/forever")
	requireStored(t, store, "recreated", "https://example.com/new")
}

// A database created by the dev branch's GORM model must keep working.
func TestSQLiteStoreOpensExistingSchema(t *testing.T) {
	file := filepath.Join(t.TempDir(), "myurls.db")
	db, err := sql.Open("sqlite", file)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE `url_mappings` (`id` integer PRIMARY KEY AUTOINCREMENT,`key` text,`value` text NOT NULL,`expires_at` integer,`created_at` integer,CONSTRAINT `uni_url_mappings_key` UNIQUE (`key`))")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO url_mappings (key, value, expires_at, created_at) VALUES ('old', 'https://example.com/old', NULL, 1)")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	store, err := newSQLiteStore(file)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	requireStored(t, store, "old", "https://example.com/old")
	claimed, err := store.Claim(context.Background(), "old", "https://example.com/new", remainingLifetime(time.Hour))
	require.NoError(t, err)
	require.False(t, claimed)
}
