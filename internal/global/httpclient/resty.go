package httpclient

import (
	"activity-punch-system/internal/global/sentry/tracing"
	"time"

	"github.com/go-resty/resty/v2"
)

var Client *resty.Client

func Init() {
	Client = resty.New().SetTimeout(10 * time.Second)

	// 配置 Sentry 性能追踪（如果 Sentry 已启用）
	if tracing.IsEnabled() {
		tracing.SetupRestyTracing(Client)
	}
}
