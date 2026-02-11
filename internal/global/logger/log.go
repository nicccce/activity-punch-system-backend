package logger

import (
	"activity-punch-system/config"
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	sentryslog "github.com/getsentry/sentry-go/slog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	instance *slog.Logger
	once     sync.Once
)

// multiHandler 组合多个 slog.Handler，将日志同时发送到多个目标
type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return newMultiHandler(handlers...)
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return newMultiHandler(handlers...)
}

// Get 获取全局 Logger 实例
func Get() *slog.Logger {
	once.Do(func() {
		cfg := config.Get()
		opts := &slog.HandlerOptions{
			AddSource: cfg.Mode == config.ModeRelease,
			Level:     getLogLevel(cfg.Log.Level),
		}

		var baseHandler slog.Handler
		if cfg.Mode == config.ModeRelease && cfg.Log.FilePath != "" {
			// 在 release 模式下输出到文件，并启用日志轮转
			lumberjackLogger := &lumberjack.Logger{
				Filename:   cfg.Log.FilePath,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   cfg.Log.Compress,
			}
			baseHandler = slog.NewJSONHandler(lumberjackLogger, opts)
		} else {
			// 在 debug 模式下（或无文件路径）输出到控制台
			baseHandler = slog.NewTextHandler(os.Stdout, opts)
		}

		var finalHandler slog.Handler = baseHandler

		// 如果配置了 Sentry DSN，添加 Sentry handler
		if cfg.Sentry.Dsn != "" {
			sentryHandler := sentryslog.Option{
				// Error 级别作为 Sentry Event 上报
				EventLevel: []slog.Level{slog.LevelError},
				// Warn 级别作为 Sentry Log 上报
				LogLevel:  []slog.Level{slog.LevelWarn, slog.LevelError},
				AddSource: cfg.Mode == config.ModeRelease,
			}.NewSentryHandler(context.Background())

			// 使用 multiHandler 组合两个 handler
			finalHandler = newMultiHandler(baseHandler, sentryHandler)
		}

		instance = slog.New(finalHandler).With(
			"app_name", "activity-punch-system",
			"env", string(cfg.Mode),
		)
	})
	return instance
}

// New 创建一个新的 Logger 实例，带模块字段
func New(module string) *slog.Logger {
	return Get().With("module", module)
}

// WithContext 从 gin.Context 中提取 client_ip 等请求信息，返回带有这些字段的 Logger
// 用于在业务日志中携带用户 IP，使其在 Sentry 日志上报中可见
func WithContext(base *slog.Logger, c interface{ ClientIP() string; GetHeader(string) string }) *slog.Logger {
	ip := c.ClientIP()
	l := base.With("client_ip", ip)

	// 优先使用代理转发的真实 IP
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		l = l.With("x_forwarded_for", forwardedFor)
	}
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		l = l.With("x_real_ip", realIP)
	}

	return l
}

// getLogLevel 将字符串级别转换为 slog.Level
func getLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
