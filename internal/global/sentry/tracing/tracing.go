// Package tracing 提供 Sentry 性能追踪的集成
// 包含 GORM、Redis 和 HTTP 客户端的追踪实现
package tracing

import (
	"activity-punch-system/config"
	"context"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// IsEnabled 检查 Sentry 追踪是否已启用
func IsEnabled() bool {
	cfg := config.Get()
	return cfg.Sentry.Dsn != ""
}

// GetSpanFromGinContext 从 gin.Context 获取当前的 Sentry span
// 如果没有活跃的 span，返回 nil
// 用于在 handler 中手动创建子 span
func GetSpanFromGinContext(c *gin.Context) *sentry.Span {
	if c == nil {
		return nil
	}

	// sentrygin 中间件会将 span 存储在 request context 中
	if c.Request != nil && c.Request.Context() != nil {
		return sentry.SpanFromContext(c.Request.Context())
	}

	return nil
}

// ContextWithSpan 返回一个包含 Sentry span 的 context
// 用于将 gin.Context 转换为可以传递给 GORM/Redis 的 context
// 用法：
//
//	ctx := tracing.ContextWithSpan(c)
//	database.DB.WithContext(ctx).Find(&users)
//	redis.RedisClient.Get(ctx, "key")
func ContextWithSpan(c *gin.Context) context.Context {
	if c == nil {
		return context.Background()
	}

	if c.Request == nil || c.Request.Context() == nil {
		return context.Background()
	}

	// sentrygin 中间件已经将 span 存储在 request context 中
	// 直接返回 request context 即可
	return c.Request.Context()
}

// StartSpan 在当前 gin.Context 的 transaction 下创建一个新的 span
// 用于追踪自定义业务逻辑
// 返回值需要调用 Finish() 方法来结束 span
// 用法：
//
//	span := tracing.StartSpan(c, "business.logic", "处理用户订单")
//	defer span.Finish()
//	// ... 业务逻辑
func StartSpan(c *gin.Context, operation, description string) *sentry.Span {
	parentSpan := GetSpanFromGinContext(c)
	if parentSpan == nil {
		// 没有父 span，创建一个 no-op span
		return &sentry.Span{}
	}

	span := parentSpan.StartChild(operation)
	span.Description = description
	return span
}

// StartSpanFromContext 从 context 创建一个新的 span
// 适用于非 gin handler 的场景
func StartSpanFromContext(ctx context.Context, operation, description string) *sentry.Span {
	parentSpan := sentry.SpanFromContext(ctx)
	if parentSpan == nil {
		// 没有父 span，创建一个 no-op span
		return &sentry.Span{}
	}

	span := parentSpan.StartChild(operation)
	span.Description = description
	return span
}
