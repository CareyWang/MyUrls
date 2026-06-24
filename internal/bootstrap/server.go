package bootstrap

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CareyWang/MyUrls/internal/handler"
	"github.com/CareyWang/MyUrls/internal/logger"
)

// initServer 初始化HTTP服务器
//
// Gin 模式已在 bootstrap.go::Run() 中根据 environment 统一设置，这里不再重复设置，
// 避免出现与 Run() 矛盾的判断逻辑。
func (a *App) initServer() {
	router := gin.Default()

	// 注册中间件
	router.Use(logger.InitServiceLogger())

	// 注册静态文件
	router.LoadHTMLGlob("web/*.html")
	router.StaticFile("/logo.png", "web/logo.png")

	// 注册路由
	a.registerRoutes(router)

	a.Server = router
}

// startServer 启动HTTP服务器
func (a *App) startServer() error {
	logger.Logger.Infof("server running on :%s", a.Config.Server.Port)
	return a.Server.Run(fmt.Sprintf(":%s", a.Config.Server.Port))
}

// registerRoutes 注册所有路由
func (a *App) registerRoutes(router *gin.Engine) {
	// 创建handler实例
	urlHandler := handler.NewURLHandler(a.Config)
	cacheHandler := handler.NewCacheHandler(a.CacheToken)

	// 注册路由
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "MyUrls",
		})
	})
	router.POST("/short", urlHandler.LongToShortHandler())
	router.GET("/:shortKey", urlHandler.ShortToLongHandler())
	
	// 注册缓存管理路由
	router.DELETE("/cache", cacheHandler.ClearCacheHandler())
	router.GET("/cache/clear", cacheHandler.ClearCacheHandler()) // 兼容GET请求便于测试
}
