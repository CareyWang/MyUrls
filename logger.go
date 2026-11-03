package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.SugaredLogger

const (
	logFilePath       = "logs/access.log"
	logFileMaxSize    = 50    // 日志文件最大大小（MB）
	logFileMaxBackups = 10    // 最多保留的备份文件数量
	logFileMaxAge     = 7     // 日志文件最长保留天数
	logFileCompress   = false // 是否压缩备份文件
)

// InitLogger 初始化 zap logger
func InitLogger() {
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

	logger = zap.New(core).Sugar()
}

// initGinLogger 初始化 gin logger，logs 目录由 lumberjack 在首次写入时创建
func initGinLogger() *zap.Logger {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
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

	return zap.New(core)
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
