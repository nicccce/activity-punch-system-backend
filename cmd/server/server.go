package server

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/httpclient"
	"activity-punch-system/internal/global/logger"
	"activity-punch-system/internal/global/middleware"
	"activity-punch-system/internal/global/redis"
	"activity-punch-system/internal/global/sentry"
	"activity-punch-system/internal/module"
	"activity-punch-system/tools"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

var log *slog.Logger

func Init() {
	config.Init()

	// 初始化 Sentry（必须在 logger 之前初始化，因为 logger 会使用 Sentry handler）
	if err := sentry.Init(); err != nil {
		// 使用标准库日志，因为 logger 还未初始化
		fmt.Printf("Sentry initialization failed: %v\n", err)
	} else if config.Get().Sentry.Dsn != "" {
		fmt.Println("Init Sentry: enabled")
	}

	// 初始化 logger（在 Sentry 之后，以便 logger 可以使用 Sentry handler）
	log = logger.New("Server")
	log.Info(fmt.Sprintf("Init Config: %s", config.Get().Mode))

	database.Init()
	log.Info(fmt.Sprintf("Init Database: %s", config.Get().Mysql.Host))

	redis.Init()
	log.Info(fmt.Sprintf("Init Redis: %s", config.Get().Redis.Host))

	httpclient.Init()
	log.Info(fmt.Sprintf("Init HttpClient: %s", config.Get().Host))

	for _, m := range module.Modules {
		log.Info(fmt.Sprintf("Init Module: %s", m.GetName()))
		m.Init()
	}
}

func Run() {
	// 确保程序退出前刷新 Sentry 缓冲区
	defer sentry.Flush(2 * time.Second)

	gin.SetMode(string(config.Get().Mode))
	r := gin.New()

	// Sentry 中间件需要在其他中间件之前添加，以便捕获所有错误
	r.Use(sentry.Middleware())
	// 将 client IP 注入 Sentry Scope，所有后续日志/事件上报自动携带 IP
	r.Use(middleware.SentryEnrichIP())

	switch config.Get().Mode {
	case config.ModeRelease:
		r.Use(middleware.Logger(logger.Get()))
	case config.ModeDebug:
		r.Use(gin.Logger())
	}
	r.Use(middleware.Cors())
	r.Use(middleware.Recovery())

	r.Static("/static/punch", "./upload/punch")

	for _, m := range module.Modules {
		log.Info(fmt.Sprintf("Init Router: %s", m.GetName()))
		m.InitRouter(r.Group("/" + config.Get().Prefix))
	}
	err := r.Run(config.Get().Host + ":" + config.Get().Port)
	tools.PanicOnErr(err)
}
