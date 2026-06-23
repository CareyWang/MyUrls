package handler

import (
	"net"
	"net/http"

	"github.com/CareyWang/MyUrls/internal/model"
	"github.com/CareyWang/MyUrls/internal/storage"
	"github.com/gin-gonic/gin"
)

// CacheHandler 处理缓存相关操作
type CacheHandler struct {
	cacheToken string
}

// NewCacheHandler 创建新的缓存处理器
func NewCacheHandler(token string) *CacheHandler {
	return &CacheHandler{
		cacheToken: token,
	}
}

// ClearCacheHandler 清空缓存接口，仅允许127.0.0.1访问
func (h *CacheHandler) ClearCacheHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := model.Response{}

		// 检查IP地址
		if !isLocalhost(c.Request) {
			resp.Code = model.ResponseCodeServerError
			resp.Msg = "access denied"
			c.JSON(http.StatusForbidden, resp)
			return
		}

		// 检查token
		token := c.Query("token")
		if token != h.cacheToken {
			resp.Code = model.ResponseCodeServerError
			resp.Msg = "invalid token"
			c.JSON(http.StatusUnauthorized, resp)
			return
		}

		// 获取清空前的缓存大小
		cache := storage.GetLRUCache()
		var sizeBefore int
		if cache != nil {
			sizeBefore = cache.Size()
		}

		// 清空LRU缓存
		storage.ClearLRUCache()

		// 获取清空后的缓存大小
		var sizeAfter int
		if cache != nil {
			sizeAfter = cache.Size()
		}

		resp.Code = model.ResponseCodeSuccess
		resp.Msg = "LRU cache cleared successfully"
		resp.Data = gin.H{
			"cleared":      true,
			"sizeBefore":   sizeBefore,
			"sizeAfter":    sizeAfter,
			"clearedCount": sizeBefore - sizeAfter,
		}
		c.JSON(http.StatusOK, resp)
	}
}

// isLocalhost 检查请求是否来自本地主机
//
// 故意不信任 X-Forwarded-For / X-Real-IP 等请求头：这些头由客户端自行设置，
// 攻击者可伪造为 127.0.0.1 绕过校验，因此只依据 TCP 连接的真实来源 RemoteAddr 判断。
func isLocalhost(r *http.Request) bool {
	ip := getClientIP(r)

	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return true
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		return parsedIP.IsLoopback()
	}

	return false
}

// getClientIP 获取 TCP 连接的真实来源地址
func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
