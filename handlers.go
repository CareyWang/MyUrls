package main

import (
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultRenewTime = time.Hour * 48 // 默认续命时间，2天

// A dot-free short key cannot shadow a static file such as /logo.png.
var shortKeyPattern = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)

func isHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// normalizeLongURL returns the long URL from plain or legacy base64 input.
func normalizeLongURL(input string) (string, bool) {
	if isHTTPURL(input) {
		return input, true
	}

	// 兼容以前的实现，这里如果是 base64 编码的字符串，进行解码
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err == nil && isHTTPURL(string(decoded)) {
		return string(decoded), true
	}
	return "", false
}

// ShortToLongHandler gets the long URL from a short URL
func ShortToLongHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := Response{}
		shortKey := c.Param("shortKey")
		longURL, err := ShortToLong(c, shortKey)
		if err != nil {
			resp.Code = ResponseCodeServerError
			resp.Msg = "failed to get long URL"
			logger.Warn("failed to get long URL: ", err.Error())

			c.JSON(500, resp)
			return
		}
		if longURL == "" {
			resp.Code = ResponseCodeServerError
			resp.Msg = "failed to get long URL, please check the short URL if exists or expired"

			c.JSON(404, resp)
			return
		}

		// todo
		// check whether need renew expiration time
		// only renew once per day
		// if err := Renew(c, shortKey, defaultRenewTime); err != nil {
		// 	logger.Warn("failed to renew short URL: ", err.Error())
		// }

		c.Redirect(301, longURL)
	}
}

type LongToShortParams struct {
	LongUrl  string `form:"longUrl" json:"longUrl" binding:"required"`
	ShortKey string `form:"shortKey" json:"shortKey" binding:"omitempty"`
}

// LongToShortHandler creates a short URL from a long URL
func LongToShortHandler(creator *shortURLCreator) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := Response{}

		// check parameters
		req := LongToShortParams{}
		if err := c.ShouldBind(&req); err != nil {
			resp.Code = ResponseCodeParamsCheckError
			resp.Msg = "invalid parameters"
			logger.Warn("invalid parameters: ", err.Error())

			c.JSON(200, resp)
			return
		}

		longURL, ok := normalizeLongURL(req.LongUrl)
		if !ok {
			resp.Code = ResponseCodeParamsCheckError
			resp.Msg = "invalid long URL, only http and https URLs are supported"

			logger.Info("invalid long URL: ", req.LongUrl)
			c.JSON(200, resp)
			return
		}

		if req.ShortKey != "" && !shortKeyPattern.MatchString(req.ShortKey) {
			resp.Code = ResponseCodeParamsCheckError
			resp.Msg = "invalid short key, only letters, digits, hyphens, and underscores are allowed"

			logger.Info("invalid short key: ", req.ShortKey)
			c.JSON(200, resp)
			return
		}

		shortKey, err := creator.Create(c.Request.Context(), longURL, req.ShortKey)
		if errors.Is(err, errShortKeyExists) {
			resp.Code = ResponseCodeParamsCheckError
			resp.Msg = "short key already exists, please use another one or leave it empty to generate automatically"

			logger.Info("short key already exists: ", req.ShortKey)
			c.JSON(200, resp)
			return
		}

		if err != nil {
			resp.Code = ResponseCodeServerError
			resp.Msg = "failed to create short URL"
			logger.Warn("failed to create short URL: ", err.Error())

			c.JSON(200, resp)
			return
		}

		shortURL := proto + "://" + domain + "/" + shortKey

		// 兼容以前的返回结构体
		respDataLegacy := gin.H{
			"Code":     ResponseCodeSuccessLegacy,
			"ShortUrl": shortURL,
		}
		c.JSON(200, respDataLegacy)
	}
}
