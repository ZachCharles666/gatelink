package api

import (
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// privateIPNets 内网 IP 段（RFC 1918 + loopback + Docker 默认网段）
var privateIPNets []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",    // loopback
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918（含 Docker 默认 172.17.0.0/16）
		"192.168.0.0/16", // RFC 1918
		"::1/128",        // IPv6 loopback
	}
	for _, cidr := range cidrs {
		_, ipNet, _ := net.ParseCIDR(cidr)
		if ipNet != nil {
			privateIPNets = append(privateIPNets, ipNet)
		}
	}
}

// Logger 请求日志中间件
// 安全原则：只记录 metadata，不记录 request body 和 response body
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		status := c.Writer.Status()

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", status).
			Float64("latency_ms", latencyMs).
			Str("client_ip", c.ClientIP()).
			Int("response_size", c.Writer.Size()).
			Msg("request")
	}
}

// Recovery 异常恢复中间件（不暴露 panic 详情给客户端）
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("panic", err).
					Str("path", c.Request.URL.Path).
					Msg("panic recovered")
				InternalError(c)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// InternalOnly 确保接口只能从内部网络访问。
// 生产模式（ENV=production）下强制校验来源 IP。
// 开发/测试模式下放通所有 IP（便于本地调试）。
func InternalOnly() gin.HandlerFunc {
	enforced := os.Getenv("ENV") == "production"

	return func(c *gin.Context) {
		if !enforced {
			c.Next()
			return
		}

		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil || !isPrivateIP(clientIP) {
			log.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("InternalOnly: blocked non-private IP")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": CodeForbidden,
				"msg":  "access denied: internal network only",
			})
			return
		}
		c.Next()
	}
}

// isPrivateIP 判断是否为内网 IP
func isPrivateIP(ip net.IP) bool {
	for _, ipNet := range privateIPNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}
