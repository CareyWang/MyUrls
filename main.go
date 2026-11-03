package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var helpFlag bool

var (
	port          = "8080"
	domain        = "localhost:8080"
	proto         = "https"
	redisAddr     = "localhost:6379"
	redisPassword = ""
	storageType   = storageTypeRedis
	sqliteFile    = "./data/myurls.db"
	cacheEnabled  = false
	cacheSize     = 128
	cacheTTL      = 300 // seconds
)

const (
	storageTypeRedis  = "redis"
	storageTypeSQLite = "sqlite"
)

func init() {
	flag.BoolVar(&helpFlag, "h", false, "display help")

	flag.StringVar(&port, "port", port, "port to run the server on")
	flag.StringVar(&domain, "domain", domain, "domain of the server")
	flag.StringVar(&proto, "proto", proto, "protocol of the server")
	flag.StringVar(&redisAddr, "conn", redisAddr, "address of the redis server")
	flag.StringVar(&redisPassword, "password", redisPassword, "password of the redis server")
	flag.StringVar(&storageType, "storage-type", storageType, "mapping storage: redis or sqlite")
	flag.StringVar(&sqliteFile, "sqlite-file", sqliteFile, "path of the sqlite database file")
	flag.BoolVar(&cacheEnabled, "cache-enabled", cacheEnabled, "cache resolved short URLs in memory")
	flag.IntVar(&cacheSize, "cache-size", cacheSize, "maximum number of cached short URLs")
	flag.IntVar(&cacheTTL, "cache-ttl", cacheTTL, "maximum seconds to cache a short URL")
}

func main() {
	flag.Parse()
	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// 从环境变量中读取配置，且环境变量优先级高于命令行参数
	if err := parseEnvirons(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	InitLogger()

	// GC optimize
	ballast := make([]byte, 1<<30) // 预分配 1G 内存，不会实际占用物理内存，不可读写该变量
	defer func() {
		logger.Infof("ballast len %v", len(ballast))
	}()

	// start http server
	if err := run(); err != nil {
		logger.Fatalln("server failed: ", err)
	}
}

func parseEnvirons() error {
	if p := os.Getenv("MYURLS_PORT"); p != "" {
		port = p
	}
	if d := os.Getenv("MYURLS_DOMAIN"); d != "" {
		domain = d
	}
	if p := os.Getenv("MYURLS_PROTO"); p != "" {
		proto = p
	}
	if c := os.Getenv("MYURLS_REDIS_CONN"); c != "" {
		redisAddr = c
	}
	if p := os.Getenv("MYURLS_REDIS_PASSWORD"); p != "" {
		redisPassword = p
	}
	if t := os.Getenv("MYURLS_STORAGE_TYPE"); t != "" {
		storageType = t
	}
	if f := os.Getenv("MYURLS_STORAGE_SQLITE_FILE"); f != "" {
		sqliteFile = f
	}
	if e := os.Getenv("MYURLS_STORAGE_CACHE_ENABLED"); e != "" {
		enabled, err := strconv.ParseBool(e)
		if err != nil {
			return fmt.Errorf("invalid MYURLS_STORAGE_CACHE_ENABLED: %w", err)
		}
		cacheEnabled = enabled
	}
	if s := os.Getenv("MYURLS_STORAGE_CACHE_SIZE"); s != "" {
		size, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid MYURLS_STORAGE_CACHE_SIZE: %w", err)
		}
		cacheSize = size
	}
	if t := os.Getenv("MYURLS_STORAGE_CACHE_TTL"); t != "" {
		ttl, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("invalid MYURLS_STORAGE_CACHE_TTL: %w", err)
		}
		cacheTTL = ttl
	}
	return nil
}

// newMappingCache returns nil when the cache is disabled.
func newMappingCache() (*resolveCache, error) {
	if !cacheEnabled {
		return nil, nil
	}
	if cacheSize <= 0 || cacheTTL <= 0 {
		return nil, fmt.Errorf("cache size and ttl must be positive, got %d and %d", cacheSize, cacheTTL)
	}
	return newResolveCache(cacheSize, time.Duration(cacheTTL)*time.Second), nil
}

func newMappingStore() (mappingStore, error) {
	switch storageType {
	case storageTypeRedis:
		return newRedisStore(redisAddr, redisPassword), nil
	case storageTypeSQLite:
		return newSQLiteStore(sqliteFile)
	default:
		return nil, fmt.Errorf("unsupported storage type %q", storageType)
	}
}

func run() error {
	// init and check storage
	cache, err := newMappingCache()
	if err != nil {
		return err
	}
	store, err := newMappingStore()
	if err != nil {
		return err
	}
	mappings := newShortURLMappings(store, cache)
	defer mappings.Close()
	if err := mappings.Ping(context.Background()); err != nil {
		return fmt.Errorf("%s ping failed: %w", storageType, err)
	}
	logger.Infof("%s ping success", storageType)

	// init and run server
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// logger
	router.Use(initServiceLogger())

	registerRoutes(router, mappings, newShortURLCreator(mappings, proto, domain))

	logger.Infof("server running on :%s", port)
	return router.Run(fmt.Sprintf(":%s", port))
}

func registerRoutes(router *gin.Engine, mappings *shortURLMappings, creator *shortURLCreator) {
	registerPublicRoutes(router)

	router.POST("/short", LongToShortHandler(creator))
	router.GET("/:shortKey", ShortToLongHandler(mappings))
}
