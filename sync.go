package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"
)

const (
	syncRedisToSQLiteCommand = "sync-redis-to-sqlite"
	defaultSyncBatchSize     = 200
)

// syncResult counts the keys seen by a sync. A skipped key is already held by a
// valid mapping in the target or vanished from the source during the sync.
type syncResult struct {
	synced, skipped, failed int
}

// runSyncRedisToSQLite copies mappings once and fails if any key could not be copied.
func runSyncRedisToSQLite(args []string) error {
	batchSize := defaultSyncBatchSize
	flags := flag.NewFlagSet(syncRedisToSQLiteCommand, flag.ContinueOnError)
	flags.StringVar(&redisAddr, "conn", redisAddr, "address of the redis server")
	flags.StringVar(&redisPassword, "password", redisPassword, "password of the redis server")
	flags.StringVar(&sqliteFile, "sqlite-file", sqliteFile, "path of the sqlite database file")
	flags.IntVar(&batchSize, "batch-size", batchSize, "number of keys to scan per batch")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if batchSize <= 0 {
		return fmt.Errorf("batch size must be positive, got %d", batchSize)
	}

	// 环境变量优先级高于命令行参数，与主程序保持一致
	if err := parseEnvirons(); err != nil {
		return err
	}
	InitLogger()

	ctx := context.Background()
	source := newRedisStore(redisAddr, redisPassword)
	defer source.Close()
	if err := source.Ping(ctx); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	target, err := newSQLiteStore(sqliteFile)
	if err != nil {
		return err
	}
	defer target.Close()

	result, err := syncMappings(ctx, source, target, batchSize)
	logger.Infof("sync finished: %d synced, %d skipped, %d failed", result.synced, result.skipped, result.failed)
	if err != nil {
		return err
	}
	if result.failed > 0 {
		return fmt.Errorf("failed to sync %d keys", result.failed)
	}
	return nil
}

// syncMappings copies every mapping from Redis, keeping each one's lifetime. It
// never overwrites a valid mapping in the target, so running it again is safe.
// Creation rules do not apply, because existing mappings stay resolvable.
func syncMappings(ctx context.Context, source *redisStore, target mappingStore, batchSize int) (syncResult, error) {
	var result syncResult
	var cursor uint64
	for {
		keys, nextCursor, err := source.client.Scan(ctx, cursor, "*", int64(batchSize)).Result()
		if err != nil {
			return result, fmt.Errorf("scan redis keys: %w", err)
		}
		for _, key := range keys {
			written, err := syncMapping(ctx, source, target, key)
			switch {
			case err != nil:
				result.failed++
				logger.Warnf("failed to sync key %q: %v", key, err)
			case written:
				result.synced++
			default:
				result.skipped++
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return result, nil
		}
	}
}

func syncMapping(ctx context.Context, source *redisStore, target mappingStore, key string) (bool, error) {
	longURL, err := source.Resolve(ctx, key)
	if errors.Is(err, errShortURLNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	remaining, err := source.TTL(ctx, key)
	if errors.Is(err, errShortURLNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var ttl time.Duration // Zero keeps a mapping without expiration permanent.
	if remaining > 0 {
		ttl = remaining
	}
	return target.Claim(ctx, key, longURL, ttl)
}
