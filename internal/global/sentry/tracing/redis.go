package tracing

import (
	"activity-punch-system/config"
	"context"
	"net"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/redis/go-redis/v9"
)

// redisStartTimeKey 用于在 context 中存储开始时间
type redisStartTimeKey struct{}

// RedisSentryHook 实现 redis.Hook 接口，用于追踪 Redis 操作
type RedisSentryHook struct {
	// slowThreshold 慢操作阈值，仅记录执行时间超过此值的操作
	// 设为 0 表示记录所有操作
	slowThreshold time.Duration
}

// NewRedisSentryHook 创建 Redis Sentry 追踪 hook
func NewRedisSentryHook() *RedisSentryHook {
	cfg := config.Get()
	threshold := time.Duration(cfg.Sentry.Tracing.RedisSlowThresholdMs) * time.Millisecond
	return &RedisSentryHook{
		slowThreshold: threshold,
	}
}

// DialHook 实现 redis.Hook 接口（可选）
func (h *RedisSentryHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook 实现 redis.Hook 接口，追踪单个 Redis 命令
func (h *RedisSentryHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// 记录开始时间
		startTime := time.Now()

		// 尝试从 context 获取当前 span
		parentSpan := sentry.SpanFromContext(ctx)

		var span *sentry.Span
		if parentSpan != nil {
			// 创建子 span
			span = parentSpan.StartChild("db.redis")
			span.Description = h.getCommandName(cmd)
			span.SetData("db.system", "redis")
			span.SetData("db.operation", cmd.Name())
			ctx = span.Context()
		}

		// 执行 Redis 命令
		err := next(ctx, cmd)

		// 计算耗时
		elapsed := time.Since(startTime)

		if span != nil {
			// 检查是否超过慢操作阈值
			if h.slowThreshold > 0 && elapsed < h.slowThreshold {
				// 未超过阈值，不发送此 span
				span.Sampled = sentry.SampledFalse
			}

			// 设置状态
			if err != nil && err != redis.Nil {
				span.Status = sentry.SpanStatusInternalError
				span.SetData("redis.error", err.Error())
			} else {
				span.Status = sentry.SpanStatusOK
			}

			span.Finish()
		}

		return err
	}
}

// ProcessPipelineHook 实现 redis.Hook 接口，追踪 Pipeline 操作
func (h *RedisSentryHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		// 记录开始时间
		startTime := time.Now()

		// 尝试从 context 获取当前 span
		parentSpan := sentry.SpanFromContext(ctx)

		var span *sentry.Span
		if parentSpan != nil {
			// 创建子 span
			span = parentSpan.StartChild("db.redis.pipeline")
			span.Description = h.getPipelineDescription(cmds)
			span.SetData("db.system", "redis")
			span.SetData("db.operation", "pipeline")
			span.SetData("redis.pipeline_length", len(cmds))
			ctx = span.Context()
		}

		// 执行 Pipeline
		err := next(ctx, cmds)

		// 计算耗时
		elapsed := time.Since(startTime)

		if span != nil {
			// 检查是否超过慢操作阈值
			if h.slowThreshold > 0 && elapsed < h.slowThreshold {
				// 未超过阈值，不发送此 span
				span.Sampled = sentry.SampledFalse
			}

			// 设置状态
			if err != nil {
				span.Status = sentry.SpanStatusInternalError
				span.SetData("redis.error", err.Error())
			} else {
				span.Status = sentry.SpanStatusOK
			}

			span.Finish()
		}

		return err
	}
}

// getCommandName 获取 Redis 命令名称
// 只返回命令名，不包含参数，避免高基数问题
func (h *RedisSentryHook) getCommandName(cmd redis.Cmder) string {
	return strings.ToUpper(cmd.Name())
}

// getPipelineDescription 获取 Pipeline 描述
func (h *RedisSentryHook) getPipelineDescription(cmds []redis.Cmder) string {
	if len(cmds) == 0 {
		return "PIPELINE (empty)"
	}
	if len(cmds) == 1 {
		return "PIPELINE: " + strings.ToUpper(cmds[0].Name())
	}
	// 只列出前几个命令，避免描述过长
	var names []string
	maxShow := 3
	for i, cmd := range cmds {
		if i >= maxShow {
			break
		}
		names = append(names, strings.ToUpper(cmd.Name()))
	}
	desc := "PIPELINE: " + strings.Join(names, ", ")
	if len(cmds) > maxShow {
		desc += "..."
	}
	return desc
}

