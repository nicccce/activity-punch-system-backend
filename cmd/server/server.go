package server

import (
	"activity-punch-system-backend/config"
	"activity-punch-system-backend/internal/global/database"
	"activity-punch-system-backend/internal/global/httpclient"
	"activity-punch-system-backend/internal/global/logger"
	"activity-punch-system-backend/internal/global/middleware"
	internalOtel "activity-punch-system-backend/internal/global/otel"
	"activity-punch-system-backend/internal/module"
	"activity-punch-system-backend/tools"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"time"
)

var log *slog.Logger

func Init() {
	config.Init()
	log = logger.New("Server")

	database.Init()

	httpclient.Init()

	if config.Get().OTel.Enable {
		log.Info("OTel Enabled")
		internalOtel.Init()
		// 确保程序退出时关闭 TracerProvider
		defer func() {
			if err := internalOtel.Shutdown(context.Background()); err != nil {
				log.Error("Failed to shutdown TracerProvider: %v", err)
			}
		}()
	}

	for _, m := range module.Modules {
		log.Info(fmt.Sprintf("Init Module: %s", m.GetName()))
		m.Init()
	}
}

func Run() {
	gin.SetMode(string(config.Get().Mode))
	r := gin.New()

	switch config.Get().Mode {
	case config.ModeRelease:
		r.Use(middleware.Logger(logger.Get()))
	case config.ModeDebug:
		r.Use(gin.Logger())
	}
	r.Use(middleware.Cors())
	r.Use(middleware.Recovery())

	if config.Get().OTel.Enable {
		r.Use(middleware.Trace())
	}

	for _, m := range module.Modules {
		log.Info(fmt.Sprintf("Init Router: %s", m.GetName()))
		m.InitRouter(r.Group("/" + config.Get().Prefix))
	}
	err := r.Run(config.Get().Host + ":" + config.Get().Port)
	tools.PanicOnErr(err)
}

func testSpan() {
	// 获取 Tracer
	tracer := otel.Tracer("test-tracer")

	// 创建一个 Span
	_, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// 模拟一些业务逻辑
	log.Info("Starting test span...")
	time.Sleep(100 * time.Millisecond) // 模拟耗时操作
	log.Info("Test span completed.")

	// 添加一些属性
	span.SetAttributes(
		attribute.String("test.key", "test-value"),
		attribute.Int("test.number", 42),
	)

	// 添加事件
	span.AddEvent("test-event", trace.WithAttributes(
		attribute.String("event.key", "event-value"),
	))
}
