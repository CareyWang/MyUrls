package main

import (
	"os"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.SugaredLogger

const (
	logFileMaxSize    = 50    // 日志文件最大大小（MB）
	logFileMaxBackups = 10    // 最多保留的备份文件数量
	logFileMaxAge     = 7     // 日志文件最长保留天数
	logFileCompress   = false // 是否压缩备份文件
)

func InitLogger() {
	// 创建 logs 目录
	createLogPath()

	// 初始化 zap logger
	initZapLogger()
}

// createLogPath 创建 logs 目录
func createLogPath() error {
	if dir, err := os.Getwd(); err == nil {
		logFilePath := dir + "/logs/"
		if err := os.MkdirAll(logFilePath, 0777); err != nil {
			panic("create log path failed: " + err.Error())
		}
	}
	return nil
}

// getLogPath 获取 logs 目录
func getLogPath() string {
	if dir, err := os.Getwd(); err == nil {
		return dir + "/logs/"
	}
	return ""
}

// 定义 zap logger
func initZapLogger() {
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

	logger = zap.New(core).Sugar()
}

// getEncoder 获取 zap encoder
func getEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
}

// initGinLogger 初始化 gin logger
func initGinLogger() *zap.Logger {
	logPath := getLogPath()
	logFileName := "access.log"

	// 日志文件
	logFile := path.Join(logPath, logFileName)

	lumberJackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    logFileMaxSize,    // 日志文件最大大小（MB）
		MaxBackups: logFileMaxBackups, // 最多保留的备份文件数量
		MaxAge:     logFileMaxAge,     // 日志文件最长保留天数
		Compress:   logFileCompress,   // 是否压缩备份文件
	}
	writeSyncer := zapcore.AddSync(lumberJackLogger)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = nil
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	core := zapcore.NewCore(encoder, writeSyncer, zapcore.InfoLevel)

	return zap.New(core, zap.AddCaller())
}

// initServiceLogger 初始化服务日志
func initServiceLogger() gin.HandlerFunc {
	_logger := initGinLogger()
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		_logger.Info(
			"request",
			zap.String("time", start.Format(time.RFC3339)),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
