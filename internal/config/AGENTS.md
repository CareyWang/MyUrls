## internal/config

### OVERVIEW
Configuration loader supporting TOML files and environment variable overrides. Implements singleton pattern with lazy initialization via GetConfig(). Defaults applied first, then file values, then env vars override both.

### WHERE TO LOOK
- config.go: Loader implementation, load order, GetConfig() global accessor
- types.go: Config struct definitions and StorageType constants

### CONVENTIONS
- Use NewLoader(path) to create loader, Load() to initialize
- Env vars use MYURLS_ prefix with uppercase and underscores (e.g., MYURLS_SERVER_PORT)
- TOML keys use lowercase with underscores (e.g., redis_addr)
- Access via GetConfig() singleton, not direct struct instantiation
- StorageType values: "redis" or "sqlite" (lowercase)

### DEFAULT VALUES
**App:** environment="production"
**Server:** port="8080", domain="localhost:8080", proto="https"
**Storage:** type="redis", redis_addr="localhost:6379", sqlite_file="./data/myurls.db", cache_enabled=true, cache_size=128, cache_ttl=300s
**Log:** level="info", format="text", output="stdout"

### ANTI-PATTERNS
- Don't access globalConfig directly, use GetConfig()
- Don't parse env vars manually, let Loader handle it
- Don't call Load() multiple times on same loader instance
- If config file missing, tolerate and continue with defaults+env
- Don't modify Config after loading, treat as immutable
- Don't skip env var prefix validation when adding new config
