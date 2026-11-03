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
	longURL, err := store.Resolve(context.Background(), shortKey)
	require.NoError(t, err)
	require.Equal(t, wantLongURL, longURL)
}

func TestStoreResolvesMissingKeyAsNotFound(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		longURL, err := store.Resolve(context.Background(), "missing")
		require.ErrorIs(t, err, errShortURLNotFound)
		require.Empty(t, longURL)
	})
}

func TestStoreClaimDoesNotOverwriteValidMapping(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		claimed, err := store.Claim(ctx, "key", "https://example.com/original", time.Hour)
		require.NoError(t, err)
		require.True(t, claimed)

		advance(59 * time.Minute)
		claimed, err = store.Claim(ctx, "key", "https://example.com/replacement", time.Hour)
		require.NoError(t, err)
		require.False(t, claimed)
		requireStored(t, store, "key", "https://example.com/original")

		// The failed claim must not have extended the original expiration.
		advance(time.Minute)
		_, err = store.Resolve(ctx, "key")
		require.ErrorIs(t, err, errShortURLNotFound)
	})
}

func TestStoreClaimReplacesExpiredMapping(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		_, err := store.Claim(ctx, "key", "https://example.com/original", time.Hour)
		require.NoError(t, err)

		advance(time.Hour)
		claimed, err := store.Claim(ctx, "key", "https://example.com/replacement", time.Hour)
		require.NoError(t, err)
		require.True(t, claimed)
		requireStored(t, store, "key", "https://example.com/replacement")
	})
}

func TestStoreZeroTTLNeverExpires(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		claimed, err := store.Claim(ctx, "key", "https://example.com/original", 0)
		require.NoError(t, err)
		require.True(t, claimed)

		advance(100 * 365 * 24 * time.Hour)
		requireStored(t, store, "key", "https://example.com/original")
		claimed, err = store.Claim(ctx, "key", "https://example.com/replacement", time.Hour)
		require.NoError(t, err)
		require.False(t, claimed)
	})
}

func TestStoreTTL(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, advance func(time.Duration)) {
		ctx := context.Background()
		_, err := store.TTL(ctx, "missing")
		require.ErrorIs(t, err, errShortURLNotFound)

		_, err = store.Claim(ctx, "expiring", "https://example.com", time.Hour)
		require.NoError(t, err)
		advance(20 * time.Minute)
		remaining, err := store.TTL(ctx, "expiring")
		require.NoError(t, err)
		require.Equal(t, 40*time.Minute, remaining)

		_, err = store.Claim(ctx, "forever", "https://example.com", 0)
		require.NoError(t, err)
		remaining, err = store.TTL(ctx, "forever")
		require.NoError(t, err)
		require.Negative(t, remaining)

		advance(40 * time.Minute)
		_, err = store.TTL(ctx, "expiring")
		require.ErrorIs(t, err, errShortURLNotFound)
	})
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
				won[i], errs[i] = store.Claim(context.Background(), "contended", fmt.Sprintf("https://example.com/%d", i), time.Hour)
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

		_, err := store.Resolve(ctx, "key")
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, errShortURLNotFound)

		claimed, err := store.Claim(ctx, "key", "https://example.com", time.Hour)
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, claimed)
	})
}

func TestMappingsRetryCollisionsOnEveryStore(t *testing.T) {
	forEachStore(t, func(t *testing.T, store mappingStore, _ func(time.Duration)) {
		ctx := context.Background()
		_, err := store.Claim(ctx, "taken", "https://example.com/original", time.Hour)
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
	claimed, err := store.Claim(context.Background(), "key", "https://example.com", time.Hour)
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, store.Close())

	reopened, err := newSQLiteStore(file)
	require.NoError(t, err)
	t.Cleanup(func() { reopened.Close() })
	requireStored(t, reopened, "key", "https://example.com")
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
	for shortKey, ttl := range map[string]time.Duration{"expired": time.Hour, "valid": 2 * time.Hour, "forever": 0} {
		_, err := store.Claim(ctx, shortKey, "https://example.com/"+shortKey, ttl)
		require.NoError(t, err)
	}

	clock.advance(time.Hour)
	// Recreating an expired key must survive the cleanup.
	_, err := store.Claim(ctx, "recreated", "https://example.com/old", time.Minute)
	require.NoError(t, err)
	clock.advance(time.Minute)
	claimed, err := store.Claim(ctx, "recreated", "https://example.com/new", time.Hour)
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
	_, err = db.Exec("CREATE TABLE `url_mappings` (`id` integer PRIMARY KEY AUTOINCREMENT,`key` text UNIQUE,`value` text NOT NULL,`expires_at` integer,`created_at` integer)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO url_mappings (key, value, expires_at, created_at) VALUES ('old', 'https://example.com/old', NULL, 1)")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	store, err := newSQLiteStore(file)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	requireStored(t, store, "old", "https://example.com/old")
	claimed, err := store.Claim(context.Background(), "old", "https://example.com/new", time.Hour)
	require.NoError(t, err)
	require.False(t, claimed)
}
