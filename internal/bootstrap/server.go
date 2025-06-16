package bootstrap

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CareyWang/MyUrls/internal/handler"
	"github.com/CareyWang/MyUrls/internal/logger"
)

// initServer 初始化HTTP服务器
func (a *App) initServer() {
	// 设置 Gin 模式
	if a.Config.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

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

	// 注册路由
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "MyUrls",
		})
	})
	router.POST("/short", urlHandler.LongToShortHandler())
	router.GET("/:shortKey", urlHandler.ShortToLongHandler())
}
