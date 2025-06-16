package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

var globalConfig *Config

// Loader 负责加载配置
type Loader struct {
	configPath string
}

// NewLoader 创建一个新的配置加载器
func NewLoader(configPath string) *Loader {
	return &Loader{configPath: configPath}
}

// Load 加载配置
func (l *Loader) Load() (*Config, error) {
	config := l.getDefaults()

	// 从文件加载
	if err := l.loadFromFile(config); err != nil {
		// 文件不存在是可接受的，但解析错误需要报告
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// 从环境变量加载，会覆盖文件中的配置
	l.loadFromEnv(config)

	globalConfig = config
	return config, nil
}

// getDefaults 设置默认配置
func (l *Loader) getDefaults() *Config {
	return &Config{
		App: AppConfig{
			Environment: "production",
		},
		Server: ServerConfig{
			Port:   "8080",
			Domain: "localhost:8080",
			Proto:  "https",
		},
		Storage: StorageConfig{
			Type:          StorageRedis,
			RedisAddr:     "localhost:6379",
			RedisPassword: "",
			SQLiteFile:    "./data/myurls.db",
			CacheEnabled:  true,
			CacheSize:     128,
			CacheTTL:      300, // 单位: 秒
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

// loadFromFile 从TOML文件加载配置
func (l *Loader) loadFromFile(config *Config) error {
	if l.configPath == "" {
		return nil // 如果未提供路径，则跳过
	}

	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		return err // 返回错误，让调用者知道文件不存在
	}

	_, err := toml.DecodeFile(l.configPath, config)
	return err
}

// loadFromEnv 从环境变量加载配置
func (l *Loader) loadFromEnv(config *Config) {
	// App
	if env := os.Getenv("MYURLS_APP_ENVIRONMENT"); env != "" {
		config.App.Environment = env
	}

	// Server
	if port := os.Getenv("MYURLS_SERVER_PORT"); port != "" {
		config.Server.Port = port
	}
	if domain := os.Getenv("MYURLS_SERVER_DOMAIN"); domain != "" {
		config.Server.Domain = domain
	}
	if proto := os.Getenv("MYURLS_SERVER_PROTO"); proto != "" {
		config.Server.Proto = proto
	}

	// Storage
	if storageType := os.Getenv("MYURLS_STORAGE_TYPE"); storageType != "" {
		config.Storage.Type = StorageType(strings.ToLower(storageType))
	}
	if redisAddr := os.Getenv("MYURLS_STORAGE_REDIS_ADDR"); redisAddr != "" {
		config.Storage.RedisAddr = redisAddr
	}
	if redisPassword := os.Getenv("MYURLS_STORAGE_REDIS_PASSWORD"); redisPassword != "" {
		config.Storage.RedisPassword = redisPassword
	}
	if sqliteFile := os.Getenv("MYURLS_STORAGE_SQLITE_FILE"); sqliteFile != "" {
		config.Storage.SQLiteFile = sqliteFile
	}
	if cacheEnabledStr := os.Getenv("MYURLS_STORAGE_CACHE_ENABLED"); cacheEnabledStr != "" {
		if enabled, err := strconv.ParseBool(cacheEnabledStr); err == nil {
			config.Storage.CacheEnabled = enabled
		}
	}
	if cacheSizeStr := os.Getenv("MYURLS_STORAGE_CACHE_SIZE"); cacheSizeStr != "" {
		if size, err := strconv.Atoi(cacheSizeStr); err == nil && size > 0 {
			config.Storage.CacheSize = size
		}
	}
	if cacheTTLStr := os.Getenv("MYURLS_STORAGE_CACHE_TTL"); cacheTTLStr != "" {
		if ttl, err := strconv.Atoi(cacheTTLStr); err == nil && ttl > 0 {
			config.Storage.CacheTTL = ttl
		}
	}

	// Log
	if level := os.Getenv("MYURLS_LOG_LEVEL"); level != "" {
		config.Log.Level = level
	}
	if format := os.Getenv("MYURLS_LOG_FORMAT"); format != "" {
		config.Log.Format = format
	}
	if output := os.Getenv("MYURLS_LOG_OUTPUT"); output != "" {
		config.Log.Output = output
	}
}

// GetConfig 返回加载的全局配置
func GetConfig() *Config {
	if globalConfig == nil {
		// 如果尚未加载，则使用默认值加载
		// 这主要用于测试或独立运行包的场景
		loader := NewLoader("")
		loader.Load()
	}
	return globalConfig
}

// GetStorageConfig 为了向后兼容，保留此函数
func GetStorageConfig() *StorageConfig {
	return &GetConfig().Storage
}
