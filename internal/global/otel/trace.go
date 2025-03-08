package otel

import (
	"activity-punch-system-backend/config"
	"activity-punch-system-backend/tools"
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

var tracerProvider *sdktrace.TracerProvider

// OTLP Exporter
func newOTLPExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithInsecure(), // 禁用 TLS
		otlptracehttp.WithEndpoint(fmt.Sprintf("%s:%s", // 设置接收端
			config.Get().OTel.AgentHost,
			config.Get().OTel.AgentPort)),
	}

	return otlptracehttp.New(ctx, opts...)
}

func Init() {
	// 启用 gRPC 调试日志
	//grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout))

	// 1. 创建资源
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.Get().OTel.ServiceName),
		),
	)
	tools.PanicOnErr(err)

	// 2. 先创建导出器
	exp, err := newOTLPExporter(context.Background())
	tools.PanicOnErr(err)

	// 3. 创建带批处理的 Span Processor
	bsp := sdktrace.NewBatchSpanProcessor(exp)

	// 4. 创建 TracerProvider（关键修复点）
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp), // 直接注入 Processor
	)

	// 5. 设置全局 TracerProvider
	otel.SetTracerProvider(tracerProvider)
}

// Shutdown 确保优雅关闭
func Shutdown(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}
