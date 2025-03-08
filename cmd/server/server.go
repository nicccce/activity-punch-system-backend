package server

import (
	"activity-punch-system-backend/config"
	"activity-punch-system-backend/internal/global/database"
	"activity-punch-system-backend/internal/global/httpclient"
	"activity-punch-system-backend/internal/global/logger"
	"activity-punch-system-backend/internal/global/middleware"
	"activity-punch-system-backend/internal/global/otel"
	"activity-punch-system-backend/internal/module"
	"activity-punch-system-backend/tools"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
)

var log *slog.Logger

func Init() {
	config.Init()
	log = logger.New("Server")

	database.Init()

	httpclient.Init()

	if config.Get().OTel.Enable {
		otel.Init()
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

	r.Use(middleware.Recovery())

	if config.Get().OTel.Enable {
		log.Info("OTel Enabled")
		r.Use(middleware.Trace())
	}

	for _, m := range module.Modules {
		log.Info(fmt.Sprintf("Init Router: %s", m.GetName()))
		m.InitRouter(r.Group("/" + config.Get().Prefix))
	}
	err := r.Run(config.Get().Host + ":" + config.Get().Port)
	tools.PanicOnErr(err)
}
