package tracing

import (
	"activity-punch-system/config"
	"net/url"

	"github.com/getsentry/sentry-go"
	"github.com/go-resty/resty/v2"
)

// spanContextKey 用于在 resty request context 中存储 span
const spanContextKey = "sentry_span"

// SetupRestyTracing 为 Resty 客户端配置 Sentry 追踪中间件
// 应在 httpclient.Init() 中调用
func SetupRestyTracing(client *resty.Client) {
	cfg := config.Get()

	// 如果未启用 HTTP 追踪，直接返回
	if !cfg.Sentry.Tracing.TraceHTTPCalls {
		return
	}

	// 添加请求前中间件：创建 span
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		ctx := req.Context()
		if ctx == nil {
			return nil
		}

		// 尝试从 context 获取当前 span
		parentSpan := sentry.SpanFromContext(ctx)
		if parentSpan == nil {
			return nil
		}

		// 创建子 span
		span := parentSpan.StartChild("http.client")
		span.Description = req.Method + " " + sanitizeURL(req.URL)
		span.SetData("http.request.method", req.Method)
		span.SetData("url.full", sanitizeURL(req.URL))

		// 添加 sentry-trace 头以支持分布式追踪
		req.SetHeader("sentry-trace", span.ToSentryTrace())
		if baggage := span.ToBaggage(); baggage != "" {
			req.SetHeader("baggage", baggage)
		}

		// 将 span 存储到 request context 中
		req.SetContext(span.Context())

		return nil
	})

	// 添加请求后中间件：结束 span
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		ctx := resp.Request.Context()
		if ctx == nil {
			return nil
		}

		// 从 context 获取 span
		span := sentry.SpanFromContext(ctx)
		if span == nil {
			return nil
		}

		// 设置响应信息
		span.SetData("http.response.status_code", resp.StatusCode())

		// 设置状态
		if resp.StatusCode() >= 400 {
			span.Status = sentry.HTTPtoSpanStatus(resp.StatusCode())
		} else {
			span.Status = sentry.SpanStatusOK
		}

		span.Finish()
		return nil
	})

	// 添加错误处理中间件
	client.OnError(func(req *resty.Request, err error) {
		if req == nil {
			return
		}

		ctx := req.Context()
		if ctx == nil {
			return
		}

		// 从 context 获取 span
		span := sentry.SpanFromContext(ctx)
		if span == nil {
			return
		}

		// 设置错误状态
		span.Status = sentry.SpanStatusInternalError
		span.SetData("http.error", err.Error())
		span.Finish()
	})
}

// sanitizeURL 清理 URL，移除敏感信息（如查询参数中的 token）
// 返回格式：scheme://host/path
func sanitizeURL(rawURL string) string {
	if rawURL == "" {
		return "unknown"
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}

	// 只返回 scheme + host + path，不包含查询参数
	result := ""
	if parsed.Scheme != "" {
		result = parsed.Scheme + "://"
	}
	if parsed.Host != "" {
		result += parsed.Host
	}
	if parsed.Path != "" {
		result += parsed.Path
	}

	if result == "" {
		return "unknown"
	}

	return result
}
