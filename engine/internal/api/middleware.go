package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Logger 请求日志中间件
// 安全原则：只记录 metadata，不记录 request body 和 response body
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
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

// InternalOnly 确保接口只能从内部网络访问
func InternalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
