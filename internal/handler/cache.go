package handler

import (
	"net"
	"net/http"
	"strings"

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
func isLocalhost(r *http.Request) bool {
	ip := getClientIP(r)

	// 检查常见的本地地址
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return true
	}

	// 解析IP地址
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		// 检查是否为环回地址
		return parsedIP.IsLoopback()
	}

	return false
}

// getClientIP 获取客户端真实IP
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.SplitSeq(xff, ",")
		for ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				return ip
			}
		}
	}

	// 检查X-Real-IP头
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 直接从RemoteAddr获取
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
