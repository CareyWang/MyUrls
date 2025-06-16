package bootstrap

import (
	"context"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/CareyWang/MyUrls/internal/config"
	"github.com/CareyWang/MyUrls/internal/logger"
	"github.com/CareyWang/MyUrls/internal/storage"
)

// App 包含应用的所有组件
type App struct {
	Config  *config.Config
	Server  *gin.Engine
	Storage storage.Driver
}

// New 创建一个新的应用实例
func New() *App {
	return &App{}
}

// Run 启动应用
func (a *App) Run() error {
	var err error

	// 1. 加载配置
	a.Config, err = a.loadConfig()
	if err != nil {
		return err
	}
	// 根据 environment 设置 GIN_MODE
	if a.Config.App.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 2. 初始化日志
	if err := a.initLogger(); err != nil {
		return err
	}

	// 3. 初始化存储
	if err := a.initStorage(); err != nil {
		return err
	}
	logger.Logger.Infof("storage (%s) ping success", a.Config.Storage.Type)

	// 4. 初始化HTTP服务器
	a.initServer()

	// 5. 启动服务
	return a.startServer()
}

// Shutdown 优雅地关闭应用
func (a *App) Shutdown(ctx context.Context) error {
	if a.Storage != nil {
		return a.Storage.Close()
	}
	return nil
}

// loadConfig 加载配置
func (a *App) loadConfig() (*config.Config, error) {
	configPath := "conf/app.toml"
	loader := config.NewLoader(configPath)
	cfg, err := loader.Load()
	if err != nil {
		// 如果配置文件不存在，这不是一个致命错误，我们可以继续使用默认值和环境变量
		if !os.IsNotExist(err) {
			logger.Logger.Fatalf("failed to load config: %v", err)
		}
	}
	return cfg, nil
}

// initLogger 初始化日志
func (a *App) initLogger() error {
	logger.Init() // 可以在这里根据 a.Config.Log 进行更详细的配置
	return nil
}

// initStorage 初始化存储
func (a *App) initStorage() error {
	if err := storage.InitStorage(&a.Config.Storage); err != nil {
		logger.Logger.Fatalln("failed to init storage:", err)
		return err
	}

	// 检查存储连接
	ctx := context.Background()
	driver := storage.GetDriver()
	if err := driver.Ping(ctx); err != nil {
		logger.Logger.Fatalln("storage ping failed:", err)
		return err
	}
	a.Storage = driver
	return nil
}
