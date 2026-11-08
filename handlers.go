package main

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ShortToLongHandler gets the long URL from a short URL
func ShortToLongHandler(mappings *shortURLMappings) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := Response{}
		shortKey := c.Param("shortKey")
		longURL, err := mappings.Resolve(c, shortKey)
		if errors.Is(err, errShortURLNotFound) {
			resp.Code = ResponseCodeServerError
			resp.Msg = "failed to get long URL, please check the short URL if exists or expired"

			c.JSON(404, resp)
			return
		}
		if err != nil {
			resp.Code = ResponseCodeServerError
			resp.Msg = "failed to get long URL"
			logger.Warn("failed to get long URL: ", err.Error())

			c.JSON(500, resp)
			return
		}

		c.Redirect(302, longURL)
	}
}

// maxShortRequestBytes bounds the creation request body before it is parsed. It
// fits the longest allowed long URL in base64 and percent-encoded form.
const maxShortRequestBytes = 256 << 10

type LongToShortParams struct {
	LongUrl  string `form:"longUrl" json:"longUrl" binding:"required"`
	ShortKey string `form:"shortKey" json:"shortKey"`
}

// LongToShortHandler creates a short URL from a long URL
func LongToShortHandler(creator *shortURLCreator) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := Response{}

		// check parameters
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxShortRequestBytes)
		req := LongToShortParams{}
		if err := c.ShouldBind(&req); err != nil {
			resp.Code = ResponseCodeParamsCheckError
			resp.Msg = "invalid parameters"
			logger.Warn("invalid parameters: ", err.Error())

			c.JSON(200, resp)
			return
		}

		shortURL, err := creator.Create(c.Request.Context(), req.LongUrl, req.ShortKey)
		var inputErr creationInputError
		if errors.As(err, &inputErr) {
			resp.Code = ResponseCodeParamsCheckError
			resp.Msg = inputErr.Error()

			logger.Infof("%s: longUrl=%s shortKey=%s", inputErr, req.LongUrl, req.ShortKey)
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

		// 兼容以前的返回结构体
		respDataLegacy := gin.H{
			"Code":     ResponseCodeSuccessLegacy,
			"ShortUrl": shortURL,
		}
		c.JSON(200, respDataLegacy)
	}
}
