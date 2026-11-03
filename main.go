package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

var helpFlag bool

var (
	port          = "8080"
	domain        = "localhost:8080"
	proto         = "https"
	redisAddr     = "localhost:6379"
	redisPassword = ""
)

func init() {
	flag.BoolVar(&helpFlag, "h", false, "display help")

	flag.StringVar(&port, "port", port, "port to run the server on")
	flag.StringVar(&domain, "domain", domain, "domain of the server")
	flag.StringVar(&proto, "proto", proto, "protocol of the server")
	flag.StringVar(&redisAddr, "conn", redisAddr, "address of the redis server")
	flag.StringVar(&redisPassword, "password", redisPassword, "password of the redis server")
}

func main() {
	flag.Parse()
	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// 从环境变量中读取配置，且环境变量优先级高于命令行参数
	parseEnvirons()

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

func parseEnvirons() {
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
}

func run() error {
	// init and check redis
	mappings := newShortURLMappings(newRedisStore(redisAddr, redisPassword))
	defer mappings.Close()
	if err := mappings.Ping(context.Background()); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	logger.Info("redis ping success")

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
