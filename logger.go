package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func InitLogger() {
	// 创建 logs 目录
	if dir, err := os.Getwd(); err == nil {
		logFilePath := dir + "/logs/"
		if err := os.MkdirAll(logFilePath, 0777); err != nil {
			panic("create log dir failed")
		}
	}

	// 初始化 zap logger
	initZapLogger()
}

// 定义 gin logger
func initLoggerForGin() *logrus.Logger {
	logFilePath := ""
	if dir, err := os.Getwd(); err == nil {
		logFilePath = dir + "/logs/"
	}
	if err := os.MkdirAll(logFilePath, 0777); err != nil {
		fmt.Println(err.Error())
	}
	logFileName := "access.log"

	// 日志文件
	fileName := path.Join(logFilePath, logFileName)
	if _, err := os.Stat(fileName); err != nil {
		if _, err := os.Create(fileName); err != nil {
			panic("create log file failed")
		}
	}

	// 写入文件
	src, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
	}

	// 实例化
	_logger := logrus.New()

	// 设置输出
	_logger.SetOutput(src)
	// logger.Out = src

	// 设置日志级别
	_logger.SetLevel(logrus.DebugLevel)

	// 设置日志格式
	_logger.Formatter = &logrus.JSONFormatter{}

	return _logger
}

// gin 文件日志
func LoggerToFile() gin.HandlerFunc {
	_logger := initLoggerForGin()
	return func(c *gin.Context) {
		logMap := make(map[string]any)

		// 开始时间
		startTime := time.Now()
		logMap["startTime"] = startTime.Format("2006-01-02 15:04:05")

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		logMap["endTime"] = endTime.Format("2006-01-02 15:04:05")

		// 执行时间
		logMap["latencyTime"] = endTime.Sub(startTime).Microseconds()

		// 请求方式
		logMap["reqMethod"] = c.Request.Method

		// 请求路由
		logMap["reqUri"] = c.Request.RequestURI

		// 状态码
		logMap["statusCode"] = c.Writer.Status()

		// 请求IP
		logMap["clientIP"] = c.ClientIP()

		// 请求 UA
		logMap["clientUA"] = c.Request.UserAgent()

		// 日志格式
		// logJson, _ := json.Marshal(logMap)
		// _logger.Info(string(logJson))

		_logger.WithFields(logrus.Fields{
			"startTime":   logMap["startTime"],
			"endTime":     logMap["endTime"],
			"latencyTime": logMap["latencyTime"],
			"reqMethod":   logMap["reqMethod"],
			"reqUri":      logMap["reqUri"],
			"statusCode":  logMap["statusCode"],
			"clientIP":    logMap["clientIP"],
			"clientUA":    logMap["clientUA"],
		}).Info()
	}
}

// 定义 zap logger
func initZapLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	_logger := zap.New(core)
	defer _logger.Sync()

	logger = _logger.Sugar()
}

func getEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
}

func getLogWriter() zapcore.WriteSyncer {
	file, _ := os.Create("./logs/runtime.log")
	return zapcore.AddSync(file)
}
