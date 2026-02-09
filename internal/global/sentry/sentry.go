package sentry

import (
	"activity-punch-system/config"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// CodedError 定义带错误码的错误接口，用于判断是否需要上报
type CodedError interface {
	error
	GetCode() int32
}

// Init 初始化 Sentry SDK
// 返回 error 如果初始化失败
func Init() error {
	cfg := config.Get()

	// 如果没有配置 DSN，跳过初始化
	if cfg.Sentry.Dsn == "" {
		return nil
	}

	// 设置性能追踪采样率（错误事件始终 100% 上报）
	tracesSampleRate := cfg.Sentry.SampleRate
	if tracesSampleRate <= 0 {
		tracesSampleRate = 1.0 // 默认 100% 采样
	}

	// 设置环境
	environment := cfg.Sentry.Environment
	if environment == "" {
		environment = string(cfg.Mode)
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.Sentry.Dsn,
		Environment:      environment,
		Release:          "activity-punch-system@1.0.0",
		SampleRate:       1.0, // 错误事件 100% 上报，不采样
		EnableTracing:    true,
		TracesSampleRate: tracesSampleRate, // 性能追踪可以采样（高流量时降低）
		EnableLogs:       true,             // 启用日志上报到 Sentry（日志也是 100% 上报）
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// 可以在这里过滤或修改事件
			return event
		},
	})

	if err != nil {
		return fmt.Errorf("sentry initialization failed: %w", err)
	}

	return nil
}

// Middleware 返回 Sentry Gin 中间件
func Middleware() gin.HandlerFunc {
	cfg := config.Get()

	// 如果没有配置 DSN，返回空中间件
	if cfg.Sentry.Dsn == "" {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return sentrygin.New(sentrygin.Options{
		Repanic:         true,  // 让 panic 继续传播，由后续的 Recovery 中间件处理
		WaitForDelivery: false, // 异步发送，不阻塞请求
		Timeout:         2 * time.Second,
	})
}

// CaptureException 捕获异常并上报到 Sentry
// 仅上报需要关注的服务器错误，不上报业务错误
func CaptureException(c *gin.Context, err error) {
	cfg := config.Get()
	if cfg.Sentry.Dsn == "" {
		return
	}

	// 检查是否是需要上报的错误类型
	if !shouldReport(err) {
		return
	}

	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			// 添加请求信息
			scope.SetRequest(c.Request)
			scope.SetTag("path", c.Request.URL.Path)
			scope.SetTag("method", c.Request.Method)

			// 添加用户信息（如果有）
			if payload, exists := c.Get("payload"); exists {
				scope.SetUser(sentry.User{
					Data: map[string]string{
						"payload": fmt.Sprintf("%+v", payload),
					},
				})
			}

			hub.CaptureException(err)
		})
	}
}

// CaptureMessage 捕获消息并上报到 Sentry
func CaptureMessage(c *gin.Context, message string) {
	cfg := config.Get()
	if cfg.Sentry.Dsn == "" {
		return
	}

	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureMessage(message)
	}
}

// shouldReport 判断错误是否需要上报到 Sentry
// 只上报服务器内部错误，不上报业务逻辑错误
func shouldReport(err error) bool {
	if e, ok := err.(CodedError); ok {
		// 只上报 5xx 错误（服务器内部错误）
		return e.GetCode() >= 500 && e.GetCode() < 600
	}
	// 非自定义错误类型，默认上报
	return true
}

// Flush 刷新 Sentry 缓冲区，确保所有事件都已发送
// 应在程序退出前调用
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}
