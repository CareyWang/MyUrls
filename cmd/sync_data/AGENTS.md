## cmd/sync_data

### OVERVIEW
CLI tool for one-shot Redis to SQLite data sync with TTL preservation. Scans all Redis keys via SCAN, batches operations, writes to SQLite, then exits.

### WHERE TO LOOK
- `main.go`: Core sync logic (DataSyncer.syncData, syncBatch, syncSingleKey)
- `main_test.go`: Table-driven tests using in-memory SQLite
- Flag/env parsing: `init()`, `parseEnvirons()` in main.go
- TTL handling: `syncSingleKey()` lines 191-200

### FLAGS & ENV OVERRIDES
Priority: flags > env vars > defaults
- `-redis-addr` (env: SYNC_REDIS_ADDR) default: localhost:6379
- `-redis-password` (env: SYNC_REDIS_PASSWORD) default: ""
- `-sqlite-file` (env: SYNC_SQLITE_FILE) default: ./data/myurls.db
- `-batch-size` default: 200 (no env override)

To add options: Add flag in `init()` + override in `parseEnvirons()`

### SYNC FLOW
1. Redis SCAN cursor iteration in batches
2. For each key: GET value + TTL
3. TTL mapping:
   - -1 (no expiry) → 10 years (10*365*24h)
   - > 0 → preserve original TTL
   - -2 (not exist) → skip
4. SQLite write via storage.SetEx()
5. Track statistics: total synced/failed/elapsed

### ANTI-PATTERNS
- Don't add `-h` flag to env vars (help-only)
- Don't modify TTL logic without updating both syncSingleKey and tests
- Don't remove batchSize from DataSyncer struct
- Don't change exit code on failure (currently fatal exit)
