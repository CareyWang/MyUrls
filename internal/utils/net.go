package utils

import (
	"net"
	"net/http"
)

// GetClientIP 获取 TCP 连接的真实来源地址
//
// 故意不信任 X-Forwarded-For / X-Real-IP 等请求头：这些头由客户端自行设置，
// 攻击者可伪造来绕过基于IP的校验或限流，因此只依据 TCP 连接的真实来源 RemoteAddr 判断。
func GetClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
