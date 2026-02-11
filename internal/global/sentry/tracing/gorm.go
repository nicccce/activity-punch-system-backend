package tracing

import (
	"activity-punch-system/config"
	"context"
	"time"

	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
)

const (
	gormSpanKey    = "sentry:span"
	gormStartKey   = "sentry:start"
	callbackPrefix = "sentry_tracing"
)

// GormTracingPlugin 实现 GORM Plugin 接口，用于追踪数据库操作
type GormTracingPlugin struct {
	// slowThreshold 慢查询阈值，仅记录执行时间超过此值的查询
	// 设为 0 表示记录所有查询
	slowThreshold time.Duration
}

// NewGormTracingPlugin 创建 GORM Sentry 追踪插件
func NewGormTracingPlugin() *GormTracingPlugin {
	cfg := config.Get()
	threshold := time.Duration(cfg.Sentry.Tracing.DBSlowThresholdMs) * time.Millisecond
	return &GormTracingPlugin{
		slowThreshold: threshold,
	}
}

// Name 返回插件名称
func (p *GormTracingPlugin) Name() string {
	return "SentryTracingPlugin"
}

// Initialize 注册 GORM 回调
func (p *GormTracingPlugin) Initialize(db *gorm.DB) error {
	// 在每个操作开始前创建 span
	_ = db.Callback().Create().Before("gorm:create").Register(callbackPrefix+":before_create", p.beforeCallback("db.sql.create"))
	_ = db.Callback().Query().Before("gorm:query").Register(callbackPrefix+":before_query", p.beforeCallback("db.sql.query"))
	_ = db.Callback().Update().Before("gorm:update").Register(callbackPrefix+":before_update", p.beforeCallback("db.sql.update"))
	_ = db.Callback().Delete().Before("gorm:delete").Register(callbackPrefix+":before_delete", p.beforeCallback("db.sql.delete"))
	_ = db.Callback().Row().Before("gorm:row").Register(callbackPrefix+":before_row", p.beforeCallback("db.sql.row"))
	_ = db.Callback().Raw().Before("gorm:raw").Register(callbackPrefix+":before_raw", p.beforeCallback("db.sql.raw"))

	// 在每个操作完成后结束 span
	_ = db.Callback().Create().After("gorm:create").Register(callbackPrefix+":after_create", p.afterCallback)
	_ = db.Callback().Query().After("gorm:query").Register(callbackPrefix+":after_query", p.afterCallback)
	_ = db.Callback().Update().After("gorm:update").Register(callbackPrefix+":after_update", p.afterCallback)
	_ = db.Callback().Delete().After("gorm:delete").Register(callbackPrefix+":after_delete", p.afterCallback)
	_ = db.Callback().Row().After("gorm:row").Register(callbackPrefix+":after_row", p.afterCallback)
	_ = db.Callback().Raw().After("gorm:raw").Register(callbackPrefix+":after_raw", p.afterCallback)

	return nil
}

// beforeCallback 在数据库操作前创建 span
func (p *GormTracingPlugin) beforeCallback(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Context == nil {
			return
		}

		ctx := db.Statement.Context

		// 记录开始时间
		db.InstanceSet(gormStartKey, time.Now())

		// 尝试从 context 获取当前 span
		parentSpan := sentry.SpanFromContext(ctx)
		if parentSpan == nil {
			// 没有父 span，不创建子 span
			return
		}

		// 创建子 span
		span := parentSpan.StartChild(operation)
		span.Description = p.getStatementDescription(db)
		span.SetData("db.system", "mysql")

		// 将 span 存储到 GORM 实例中
		db.InstanceSet(gormSpanKey, span)
		// 更新 context 以便子操作可以继续追踪
		db.Statement.Context = span.Context()
	}
}

// afterCallback 在数据库操作后结束 span
func (p *GormTracingPlugin) afterCallback(db *gorm.DB) {
	if db.Statement == nil {
		return
	}

	// 获取开始时间
	startVal, ok := db.InstanceGet(gormStartKey)
	if !ok {
		return
	}
	startTime, ok := startVal.(time.Time)
	if !ok {
		return
	}

	elapsed := time.Since(startTime)

	// 获取 span
	spanVal, ok := db.InstanceGet(gormSpanKey)
	if !ok {
		return
	}
	span, ok := spanVal.(*sentry.Span)
	if !ok || span == nil {
		return
	}

	// 检查是否超过慢查询阈值
	if p.slowThreshold > 0 && elapsed < p.slowThreshold {
		// 未超过阈值，丢弃此 span（不发送）
		// 注意：sentry-go 没有直接丢弃 span 的方法，
		// 但我们可以通过设置 Sampled = false 来避免发送
		span.Sampled = sentry.SampledFalse
	}

	// 添加额外信息
	span.SetData("db.rows_affected", db.RowsAffected)
	if db.Error != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("db.error", db.Error.Error())
	} else {
		span.Status = sentry.SpanStatusOK
	}

	// 完成 span
	span.Finish()
}

// getStatementDescription 获取 SQL 语句描述
// 使用表名作为描述，避免高基数问题
func (p *GormTracingPlugin) getStatementDescription(db *gorm.DB) string {
	if db.Statement == nil {
		return "unknown"
	}
	// 使用表名作为描述，避免记录完整 SQL（可能包含敏感数据）
	table := db.Statement.Table
	if table == "" {
		return "unknown"
	}
	return table
}

// WithSentryContext 为 GORM 查询添加 Sentry context
// 用法：database.DB.WithContext(tracing.WithSentryContext(c)).Find(&users)
// 注意：该函数现在只是简单地返回传入的 context，
// 因为 span 信息已经在 context 中通过 sentry.SpanFromContext 可以获取
func WithSentryContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	// context 中已经包含了 span 信息，直接返回即可
	return ctx
}
