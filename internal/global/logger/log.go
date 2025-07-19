package logger

import (
	"activity-punch-system/config"
	"log/slog"
	"os"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	instance *slog.Logger
	once     sync.Once
)

// Get 获取全局 Logger 实例
func Get() *slog.Logger {
	once.Do(func() {
		cfg := config.Get()
		opts := &slog.HandlerOptions{
			AddSource: cfg.Mode == config.ModeRelease,
			Level:     getLogLevel(cfg.Log.Level),
		}

		var handler slog.Handler
		if cfg.Mode == config.ModeRelease && cfg.Log.FilePath != "" {
			// 在 release 模式下输出到文件，并启用日志轮转
			lumberjackLogger := &lumberjack.Logger{
				Filename:   cfg.Log.FilePath,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   cfg.Log.Compress,
			}
			handler = slog.NewJSONHandler(lumberjackLogger, opts)
		} else {
			// 在 debug 模式下（或无文件路径）输出到控制台
			handler = slog.NewTextHandler(os.Stdout, opts)
		}

		instance = slog.New(handler).With(
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
