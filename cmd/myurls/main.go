package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/CareyWang/MyUrls/internal/config"
	"github.com/CareyWang/MyUrls/internal/handler"
	"github.com/CareyWang/MyUrls/internal/logger"
	"github.com/CareyWang/MyUrls/internal/storage"
)

var helpFlag bool

var (
	port   = "8080"
	domain = "localhost:8080"
	proto  = "https"
)

func init() {
	flag.BoolVar(&helpFlag, "h", false, "display help")

	flag.StringVar(&port, "port", port, "port to run the server on")
	flag.StringVar(&domain, "domain", domain, "domain of the server")
	flag.StringVar(&proto, "proto", proto, "protocol of the server")
}

func main() {
	flag.Parse()
	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// 从环境变量中读取配置，且环境变量优先级高于命令行参数
	parseEnvirons()

	// 更新handler中的全局变量
	handler.Domain = domain
	handler.Proto = proto

	logger.Init()

	// 初始化存储驱动
	storageConfig := config.GetStorageConfig()
	if err := storage.InitStorage(storageConfig); err != nil {
		logger.Logger.Fatalln("failed to init storage:", err)
	}

	// 检查存储连接
	ctx := context.Background()
	driver := storage.GetDriver()
	if err := driver.Ping(ctx); err != nil {
		logger.Logger.Fatalln("storage ping failed:", err)
	}
	logger.Logger.Infof("storage (%s) ping success", storageConfig.Type)

	// GC optimize
	ballast := make([]byte, 1<<30) // 预分配 1G 内存，不会实际占用物理内存，不可读写该变量
	defer func() {
		logger.Logger.Info("ballast len %v", len(ballast))
		driver.Close() // 关闭存储连接
	}()

	// start http server
	run()
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
}

func run() {
	// init and run server
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// logger
	router.Use(logger.InitServiceLogger())

	// static files
	router.LoadHTMLGlob("web/*.html")
	router.StaticFile("/logo.png", "web/logo.png")

	router.GET("/", func(context *gin.Context) {
		context.HTML(http.StatusOK, "index.html", gin.H{
			"title": "MyUrls",
		})
	})

	router.POST("/short", handler.LongToShortHandler())
	router.GET("/:shortKey", handler.ShortToLongHandler())

	logger.Logger.Infof("server running on :%s", port)
	router.Run(fmt.Sprintf(":%s", port))
}
