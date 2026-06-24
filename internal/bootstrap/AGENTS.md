## internal/bootstrap

### OVERVIEW
Application bootstrap orchestrates startup sequence and HTTP server initialization. Handles component lifecycle from config loading through graceful shutdown.

### WHERE TO LOOK
- **bootstrap.go**: App struct, Run() init sequence, Shutdown()
- **server.go**: HTTP server setup, route registration, gin mode

### INIT SEQUENCE
Run() executes components in order:
1. Load config (conf/app.toml → env vars → defaults)
2. Set gin mode (DebugMode if development, ReleaseMode otherwise)
3. Initialize logger
4. Initialize storage (Redis/SQLite driver with Ping check)
5. Generate cache token for cache clear endpoint
6. Initialize HTTP server (gin.Default, middleware, templates, routes)
7. Start HTTP server on configured port

### CONVENTIONS
- Gin mode set once in Run(), before initServer() runs
- gin.ReleaseMode unless environment == "development"
- HTTP server uses gin.Default() with service logger middleware
- HTML templates loaded from web/*.html, static logo at web/logo.png
- Handlers instantiated with config or cache token during route registration
- Routes registered: / (index), POST /short (rate-limited), GET /:shortKey, DELETE /cache, GET /cache/clear
- Storage driver accessible via App.Storage after initialization
- Cache token logged for debugging purposes

### ANTI-PATTERNS
- Do not set gin mode after Run() - set once in bootstrap
- Do not register routes before loading templates or middleware
- Do not access App.Storage before initStorage() completes
- Do not start server without successful storage Ping check
- Do not skip cache token generation - required for /cache endpoints
