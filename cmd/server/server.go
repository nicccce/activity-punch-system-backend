package server

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/httpclient"
	"activity-punch-system/internal/global/logger"
	"activity-punch-system/internal/global/middleware"
	"activity-punch-system/internal/global/redis"
	"activity-punch-system/internal/module"
	"activity-punch-system/tools"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
)

var log *slog.Logger

func Init() {
	config.Init()
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

	for _, m := range module.Modules {
		log.Info(fmt.Sprintf("Init Router: %s", m.GetName()))
		m.InitRouter(r.Group("/" + config.Get().Prefix))
	}
	err := r.Run(config.Get().Host + ":" + config.Get().Port)
	tools.PanicOnErr(err)
}
