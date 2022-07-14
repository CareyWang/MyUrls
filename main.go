package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

type Response struct {
	Code     int
	Message  string
	LongUrl  string
	ShortUrl string
}

type redisPoolConf struct {
	maxIdle        int
	maxActive      int
	maxIdleTimeout int
	host           string
	password       string
	db             int
	handleTimeout  int
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const defaultPort int = 8002
const defaultExpire = 180
const defaultRedisConfig = "127.0.0.1:6379"

const defaultLockPrefix = "myurls:lock:"
const defaultRenewal = 1

const secondsPerDay = 24 * 3600

var redisPool *redis.Pool
var redisPoolConfig *redisPoolConf
var redisClient redis.Conn

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Log 收集中间件
	router.Use(LoggerToFile())

	router.LoadHTMLGlob("public/*.html")

	port := flag.Int("port", defaultPort, "服务端口")
	domain := flag.String("domain", "", "短链接域名，必填项")
	ttl := flag.Int("ttl", defaultExpire, "短链接有效期，单位(天)，默认180天。")
	conn := flag.String("conn", defaultRedisConfig, "Redis连接，格式: host:port")
	passwd := flag.String("passwd", "", "Redis连接密码")
	https := flag.Int("https", 1, "是否返回 https 短链接")
	flag.Parse()

	if *domain == "" {
		flag.Usage()
		log.Fatalln("缺少关键参数")
	}

	redisPoolConfig = &redisPoolConf{
		maxIdle:        1024,
		maxActive:      1024,
		maxIdleTimeout: 30,
		host:           *conn,
		password:       *passwd,
		db:             0,
		handleTimeout:  30,
	}
	initRedisPool()

	router.GET("/", func(context *gin.Context) {
		context.HTML(http.StatusOK, "index.html", gin.H{
			"title": "MyUrls",
		})
	})

	router.POST("/short", func(context *gin.Context) {
		res := &Response{
			Code:     1,
			Message:  "",
			LongUrl:  "",
			ShortUrl: "",
		}

		longUrl := context.PostForm("longUrl")
		shortKey := context.PostForm("shortKey")
		if longUrl == "" {
			res.Code = 0
			res.Message = "longUrl为空"
			context.JSON(200, *res)
			return
		}

		_longUrl, _ := base64.StdEncoding.DecodeString(longUrl)
		longUrl = string(_longUrl)
		res.LongUrl = longUrl

		// 根据有没有填写 short key，分别执行
		if shortKey != "" {
			redisClient := redisPool.Get()

			// 检测短链是否已存在
			_exists, _ := redis.String(redisClient.Do("get", shortKey))
			if _exists != "" && _exists != longUrl {
				res.Code = 0
				res.Message = "短链接已存在，请更换key"
				context.JSON(200, *res)
				return
			}

			// 存储
			_, _ = redisClient.Do("set", shortKey, longUrl)

		} else {
			shortKey = longToShort(longUrl, *ttl*secondsPerDay)
		}

		protocol := "http://"
		if *https != 0 {
			protocol = "https://"
		}
		res.ShortUrl = protocol + *domain + "/" + shortKey

		// context.Header("Access-Control-Allow-Origin", "*")
		context.JSON(200, *res)
	})

	router.GET("/:shortKey", func(context *gin.Context) {
		shortKey := context.Param("shortKey")
		longUrl := shortToLong(shortKey)

		if longUrl == "" {
			context.String(http.StatusNotFound, "短链接不存在或已过期")
		} else {
			context.Redirect(http.StatusMovedPermanently, longUrl)
		}
	})

	router.Run(fmt.Sprintf(":%d", *port))
}

// 短链接转长链接
func shortToLong(shortKey string) string {
	redisClient = redisPool.Get()
	defer redisClient.Close()

	longUrl, _ := redis.String(redisClient.Do("get", shortKey))

	// 获取到长链接后，续命1天。每天仅允许续命1次。
	if longUrl != "" {
		renew(shortKey)
	}

	return longUrl
}

// 长链接转短链接
func longToShort(longUrl string, ttl int) string {
	redisClient = redisPool.Get()
	defer redisClient.Close()

	// 是否生成过该长链接对应短链接
	_existsKey, _ := redis.String(redisClient.Do("get", longUrl))
	if _existsKey != "" {
		_, _ = redisClient.Do("expire", _existsKey, ttl)

		log.Println("Hit cache: " + _existsKey)
		return _existsKey
	}

	// 重试三次
	var shortKey string
	for i := 0; i < 3; i++ {
		shortKey = generate(7)

		_existsLongUrl, _ := redis.String(redisClient.Do("get", shortKey))
		if _existsLongUrl == "" {
			break
		}
	}

	if shortKey != "" {
		_, _ = redisClient.Do("mset", shortKey, longUrl, longUrl, shortKey)

		_, _ = redisClient.Do("expire", shortKey, ttl)
		_, _ = redisClient.Do("expire", longUrl, secondsPerDay)
	}

	return shortKey
}

// 续命
func renew(shortKey string) {
	redisClient = redisPool.Get()
	defer redisClient.Close()

	// 加锁
	lockKey := defaultLockPrefix + shortKey
	lock, _ := redis.Int(redisClient.Do("setnx", lockKey, 1))
	if lock == 1 {
		// 设置锁过期时间
		_, _ = redisClient.Do("expire", lockKey, defaultRenewal*secondsPerDay)

		// 续命
		ttl, err := redis.Int(redisClient.Do("ttl", shortKey))
		if err == nil && ttl != -1 {
			_, _ = redisClient.Do("expire", shortKey, ttl+defaultRenewal*secondsPerDay)
		}
	}
}

// 产生一个63位随机整数
func generate(bits int) string {
	b := make([]byte, bits)

	currentTime := time.Now().UnixNano()
	rand.Seed(currentTime)

	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// 定义 logger
func Logger() *logrus.Logger {
	logFilePath := ""
	if dir, err := os.Getwd(); err == nil {
		logFilePath = dir + "/logs/"
	}
	if err := os.MkdirAll(logFilePath, 0777); err != nil {
		fmt.Println(err.Error())
	}
	logFileName := "access.log"

	//日志文件
	fileName := path.Join(logFilePath, logFileName)
	if _, err := os.Stat(fileName); err != nil {
		if _, err := os.Create(fileName); err != nil {
			fmt.Println(err.Error())
		}
	}

	//写入文件
	src, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
	}

	//实例化
	logger := logrus.New()

	//设置输出
	logger.SetOutput(src)
	// logger.Out = src

	//设置日志级别
	logger.SetLevel(logrus.DebugLevel)

	//设置日志格式
	logger.Formatter = &logrus.JSONFormatter{}

	return logger
}

// 文件日志
func LoggerToFile() gin.HandlerFunc {
	logger := Logger()
	return func(c *gin.Context) {
		logMap := make(map[string]interface{})

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

		//日志格式
		// logJson, _ := json.Marshal(logMap)
		// logger.Info(string(logJson))

		logger.WithFields(logrus.Fields{
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

// redis 连接池
func initRedisPool() {
	// 建立连接池
	redisPool = &redis.Pool{
		MaxIdle:     redisPoolConfig.maxIdle,
		MaxActive:   redisPoolConfig.maxActive,
		IdleTimeout: time.Duration(redisPoolConfig.maxIdleTimeout) * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			con, err := redis.Dial("tcp", redisPoolConfig.host,
				redis.DialPassword(redisPoolConfig.password),
				redis.DialDatabase(redisPoolConfig.db),
				redis.DialConnectTimeout(time.Duration(redisPoolConfig.handleTimeout)*time.Second),
				redis.DialReadTimeout(time.Duration(redisPoolConfig.handleTimeout)*time.Second),
				redis.DialWriteTimeout(time.Duration(redisPoolConfig.handleTimeout)*time.Second))
			if err != nil {
				return nil, err
			}
			return con, nil
		},
	}
}
