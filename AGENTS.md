# PROJECT KNOWLEDGE BASE

**Generated:** 2026-01-13 11:10:17 +0800
**Commit:** 1757a48
**Branch:** dev

## OVERVIEW
Go-based local URL shortener with Redis/SQLite backends, Gin HTTP layer, and an LRU cache. Two binaries: main service and a sync_data CLI for Redis to SQLite sync.

## STRUCTURE
```
./
├── cmd/                 # Entry points (myurls, sync_data)
├── internal/            # App layers
├── conf/                # TOML config (app.toml)
├── web/                 # Static web UI assets
├── scripts/             # Build scripts per platform
├── build/               # Build outputs
├── data/                # SQLite/db runtime data
└── logs/                # Runtime logs
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| HTTP routes, request/response | internal/handler | Gin handlers, response shapes, status codes |
| Business logic | internal/service | ShortToLong, LongToShort, Renew, CheckKeyExists |
| Storage drivers | internal/storage | Redis/SQLite drivers, LRU cache, manager |
| Config loading | internal/config | Defaults, TOML + env overrides |
| App bootstrap | internal/bootstrap | Startup flow, server init, route wiring |
| CLI sync tool | cmd/sync_data | Redis to SQLite one-shot sync |
| Main binary entry | cmd/myurls/main.go | bootstrap.New().Run() |

## CODE MAP
| Symbol | Type | Location | Refs | Role |
|--------|------|----------|------|------|
| main | Function | cmd/myurls/main.go:9 | - | Service entrypoint |
| main | Function | cmd/sync_data/main.go:31 | - | Sync CLI entrypoint |
| DataSyncer | Struct | cmd/sync_data/main.go:102 | - | Batch sync orchestrator |
| Driver | Interface | internal/storage/interface.go:9 | - | Storage contract |

## CONVENTIONS
- Error handling: service layer returns empty string on error; handlers return HTTP codes.
- Config priority: defaults → TOML (conf/app.toml) → env vars (MYURLS_ prefix).
- Comments: Chinese for business logic, English for technical details.
- Imports: stdlib → third-party → github.com/CareyWang/MyUrls.

## ANTI-PATTERNS (THIS PROJECT)
- Do not bypass service layer in handlers.
- Do not access storage driver before InitStorage.
- Do not change TTL behavior without updating tests.

## UNIQUE STYLES
- LRU cache with TTL and background cleanup (internal/storage/lru.go).
- SQLite driver simulates Redis "redis: nil" error for missing keys.

## COMMANDS
```bash
make default
make all
make sync_data
make fmt
go test ./...
SKIP_REDIS_TESTS=1 go test ./...
```

## NOTES
- CI uses actions/checkout@master in workflows (non-standard but current).
- Cache clear endpoint requires localhost IP + token.
