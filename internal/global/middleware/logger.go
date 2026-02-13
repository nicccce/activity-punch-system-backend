package middleware

import (
	"bytes"
	"log/slog"
	"time"

	sentrylib "github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// maxResponseLogSize 日志中记录的响应体最大大小（10KB）
const maxResponseLogSize = 10 * 1024

// responseBodyWriter 包装 gin.ResponseWriter 以捕获响应体内容
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.body.Len() < maxResponseLogSize {
		// 只缓存前 maxResponseLogSize 字节，避免大响应占用过多内存
		remaining := maxResponseLogSize - w.body.Len()
		if len(b) <= remaining {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remaining])
		}
	}
	return w.ResponseWriter.Write(b)
}

func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 包装 ResponseWriter 以捕获响应体
		blw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 获取响应体（截断处理）
		responseBody := blw.body.String()
		if len(responseBody) > maxResponseLogSize {
			responseBody = responseBody[:maxResponseLogSize] + "...(truncated)"
		}

		// 记录请求日志（包含响应体，Info 级别会上报到 Sentry Log）
		log.Info("HTTP Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"status", c.Writer.Status(),
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
			"response_body", responseBody,
		)
	}
}

// SentryEnrichIP 中间件：将 client IP 注入到 Sentry Scope 中
// 放在 sentry.Middleware() 之后，所有后续的 Sentry 上报（日志、事件、性能追踪）都会自动携带 IP
func SentryEnrichIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if hub := sentrygin.GetHubFromContext(c); hub != nil {
			hub.ConfigureScope(func(scope *sentrylib.Scope) {
				clientIP := c.ClientIP()

				// 将 IP 设置到 Sentry User 中
				scope.SetUser(sentrylib.User{
					IPAddress: clientIP,
				})

				// 同时作为 Tag 方便在 Sentry 中搜索和过滤
				scope.SetTag("client_ip", clientIP)

				// 记录代理转发的真实 IP
				if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
					scope.SetTag("x_forwarded_for", forwardedFor)
				}
				if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
					scope.SetTag("x_real_ip", realIP)
				}
			})
		}
		c.Next()
	}
}
