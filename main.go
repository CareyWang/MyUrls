package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

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
	if len(os.Args) > 1 && os.Args[1] == syncRedisToSQLiteCommand {
		if err := runSyncRedisToSQLite(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	flag.Parse()

	// 从环境变量中读取配置，且环境变量优先级高于命令行参数
	if err := parseEnvirons(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	InitLogger()

	// start http server, stopping it gracefully on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		logger.Fatalln("server failed: ", err)
	}
}

// envFlags maps each environment variable to the flag it overrides.
var envFlags = []struct{ env, flag string }{
	{"MYURLS_PORT", "port"},
	{"MYURLS_DOMAIN", "domain"},
	{"MYURLS_PROTO", "proto"},
	{"MYURLS_REDIS_CONN", "conn"},
	{"MYURLS_REDIS_PASSWORD", "password"},
	{"MYURLS_STORAGE_TYPE", "storage-type"},
	{"MYURLS_STORAGE_SQLITE_FILE", "sqlite-file"},
	{"MYURLS_STORAGE_CACHE_ENABLED", "cache-enabled"},
	{"MYURLS_STORAGE_CACHE_SIZE", "cache-size"},
	{"MYURLS_STORAGE_CACHE_TTL", "cache-ttl"},
}

func parseEnvirons() error {
	for _, e := range envFlags {
		if value := os.Getenv(e.env); value != "" {
			if err := flag.Set(e.flag, value); err != nil {
				return fmt.Errorf("invalid %s: %w", e.env, err)
			}
		}
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

// shutdownTimeout bounds how long in-flight requests may finish after a stop signal.
const shutdownTimeout = 10 * time.Second

// run serves until ctx is done, then drains requests and closes the storage.
func run(ctx context.Context) error {
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

	server := &http.Server{Addr: ":" + port, Handler: router}
	served := make(chan error, 1)
	go func() { served <- server.ListenAndServe() }()
	logger.Infof("server running on :%s", port)

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func registerRoutes(router *gin.Engine, mappings *shortURLMappings, creator *shortURLCreator) {
	registerPublicRoutes(router)

	router.POST("/short", LongToShortHandler(creator))
	router.GET("/:shortKey", ShortToLongHandler(mappings))
}
