package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"math/rand"
	"net/http"
)

type Response struct {
	Code     int
	Message  string
	LongUrl  string
	ShortUrl string
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const defaultPort int = 8002
const defaultExpire = 90
const redisConfig = "127.0.0.1:6379"

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	port := flag.Int("port", defaultPort, "服务端口")
	domain := flag.String("domain", "", "短链接域名，必填项")
	ttl := flag.Int("ttl", defaultExpire, "短链接有效期，单位(天)，默认90天。")
	flag.Parse()

	if *domain == "" {
		flag.Usage()
		return
	}

	router.POST("/short", func(context *gin.Context) {
		res := &Response{
			Code:     1,
			Message:  "",
			LongUrl:  "",
			ShortUrl: "",
		}

		longUrl := context.PostForm("longUrl")
		_longUrl, _ := base64.StdEncoding.DecodeString(longUrl)
		longUrl = string(_longUrl)

		shortKey := longToShort(longUrl, *ttl * 24 * 3600)
		if shortKey == "" {
			res.Code = 0
			res.Message = "短链接生成失败"
			context.JSON(500, *res)
			return
		}

		res.LongUrl = longUrl
		res.ShortUrl = "http://" + *domain + "/" + shortKey
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
	redisClient := initRedis()

	longUrl, _ := redis.String(redisClient.Do("get", shortKey))
	return longUrl
}

// 长链接转短链接
func longToShort(longUrl string, ttl int) string {
	redisClient := initRedis()

	// 是否生成过该长链接对应短链接
	_existsKey, _ := redis.String(redisClient.Do("get", longUrl))
	if _existsKey != "" {
		_, _ = redisClient.Do("expire", _existsKey, ttl)

		return _existsKey
	}

	// 重试三次
	var shortKey string
	for i := 0; i < 3; i++ {
		shortKey = generate(6)

		_existsLongUrl, _ := redis.String(redisClient.Do("get", longUrl))
		if _existsLongUrl != "" {
			break
		}
	}

	if shortKey != "" {
		_, _ = redisClient.Do("mset", shortKey, longUrl, longUrl, shortKey)

		_, _ = redisClient.Do("expire", shortKey, ttl)
		_, _ = redisClient.Do("expire", longUrl, ttl)
	}

	return shortKey
}

// 产生一个63位随机整数，除以字符数取余获取对应字符
func generate(bits int) string {
	b := make([]byte, bits)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func initRedis() redis.Conn {
	client, _ := redis.Dial("tcp", redisConfig)

	return client
}
