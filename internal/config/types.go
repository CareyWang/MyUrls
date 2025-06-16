package config

type StorageType string

const (
	StorageRedis  StorageType = "redis"
	StorageSQLite StorageType = "sqlite"
)

type Config struct {
	App     AppConfig     `toml:"app"`
	Server  ServerConfig  `toml:"server"`
	Storage StorageConfig `toml:"storage"`
	Log     LogConfig     `toml:"log"`
}

type AppConfig struct {
	Environment string `toml:"environment"`
}

type ServerConfig struct {
	Port   string `toml:"port"`
	Domain string `toml:"domain"`
	Proto  string `toml:"proto"`
}

type StorageConfig struct {
	Type          StorageType `toml:"type"`
	RedisAddr     string      `toml:"redis_addr"`
	RedisPassword string      `toml:"redis_password"`
	SQLiteFile    string      `toml:"sqlite_file"`
	CacheEnabled  bool        `toml:"cache_enabled"`
	CacheSize     int         `toml:"cache_size"`
	CacheTTL      int         `toml:"cache_ttl"` // 单位：秒
}

type LogConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
	Output string `toml:"output"`
}
