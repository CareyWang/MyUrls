package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CareyWang/MyUrls/internal/config"
	"github.com/CareyWang/MyUrls/internal/logger"
	"github.com/CareyWang/MyUrls/internal/model"
	"github.com/CareyWang/MyUrls/internal/service"
	"github.com/CareyWang/MyUrls/internal/utils"
)

const defaultTTL = time.Hour * 24 * 365 // 默认过期时间，1年
const defaultRenewTime = time.Hour * 48 // 默认续命时间，2天
const defaultShortKeyLength = 7         // 默认短链接长度，7位

// URLHandler 负责处理URL相关的请求
type URLHandler struct {
	Config *config.Config
}

// NewURLHandler 创建一个新的URLHandler
func NewURLHandler(cfg *config.Config) *URLHandler {
	return &URLHandler{Config: cfg}
}

// ShortToLongHandler gets the long URL from a short URL
func (h *URLHandler) ShortToLongHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := model.Response{}
		shortKey := c.Param("shortKey")
		longURL := service.ShortToLong(c, shortKey)
		if longURL == "" {
			resp.Code = model.ResponseCodeServerError
			resp.Msg = "failed to get long URL, please check the short URL if exists or expired"

			c.JSON(404, resp)
			return
		}

		// todo
		// check whether need renew expiration time
		// only renew once per day
		// if err := service.Renew(c, shortKey, defaultRenewTime); err != nil {
		// 	logger.Logger.Warn("failed to renew short URL: ", err.Error())
		// }

		// 302临时重定向：短链有TTL过期机制，若使用301浏览器会永久缓存跳转目标，
		// 导致key过期或被复用后客户端仍跳转到旧地址
		c.Redirect(http.StatusFound, longURL)
	}
}

type LongToShortParams struct {
	LongUrl  string `form:"longUrl" json:"longUrl" binding:"required"`
	ShortKey string `form:"shortKey" json:"shortKey" binding:"omitempty"`
}

// LongToShortHandler creates a short URL from a long URL
func (h *URLHandler) LongToShortHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := model.Response{}

		// check parameters
		req := LongToShortParams{}
		if err := c.ShouldBind(&req); err != nil {
			resp.Code = model.ResponseCodeParamsCheckError
			resp.Msg = "invalid parameters"
			logger.Logger.Warn("invalid parameters: ", err.Error())

			c.JSON(200, resp)
			return
		}

		// 兼容以前的实现，这里如果是 base64 编码的字符串，进行解码
		_longUrl, err := base64.StdEncoding.DecodeString(req.LongUrl)
		if err == nil {
			req.LongUrl = string(_longUrl)
		}

		// 校验长链接，拒绝 javascript:/data: 等危险scheme，避免服务被用作钓鱼/开放重定向跳板
		if err := utils.ValidateLongURL(req.LongUrl); err != nil {
			resp.Code = model.ResponseCodeParamsCheckError
			resp.Msg = "invalid long url: " + err.Error()
			logger.Logger.Warn("invalid long url: ", err.Error())

			c.JSON(200, resp)
			return
		}

		// 校验自定义短链接key，字符集与自动生成key一致，长度不超过32位
		if err := utils.ValidateShortKey(req.ShortKey); err != nil {
			resp.Code = model.ResponseCodeParamsCheckError
			resp.Msg = "invalid short key: " + err.Error()
			logger.Logger.Warn("invalid short key: ", err.Error())

			c.JSON(200, resp)
			return
		}

		// 原子地创建短链：key存在性检查与写入在存储层合并为单次操作，避免并发下的竞态覆盖
		shortKey, err := service.CreateShort(c, req.ShortKey, req.LongUrl, defaultTTL, defaultShortKeyLength)
		if err != nil {
			if errors.Is(err, service.ErrShortKeyExists) {
				resp.Code = model.ResponseCodeParamsCheckError
				resp.Msg = "short key already exists, please use another one or leave it empty to generate automatically"
				logger.Logger.Info("short key already exists: ", req.ShortKey)
			} else {
				resp.Code = model.ResponseCodeServerError
				resp.Msg = "failed to create short URL"
				logger.Logger.Warn("failed to create short URL: ", err.Error())
			}

			c.JSON(200, resp)
			return
		}

		shortURL := h.Config.Server.Proto + "://" + h.Config.Server.Domain + "/" + shortKey

		// 兼容以前的返回结构体
		respDataLegacy := gin.H{
			"Code":     model.ResponseCodeSuccessLegacy,
			"ShortUrl": shortURL,
		}
		c.JSON(200, respDataLegacy)
	}
}
