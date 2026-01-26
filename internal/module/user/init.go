package user

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/logger"
	sduauth "github.com/nicccce/golang-sdu-auth"
	"log/slog"
)

var log *slog.Logger

type ModuleUser struct{}

func (u *ModuleUser) GetName() string {
	return "User"
}

var casClient *sduauth.CASClient

func (u *ModuleUser) Init() {
	log = logger.New("User")
	casClient, _ = sduauth.NewCASClient(config.Get().Sdulogin.CasKey)
	//println("key:", config.Get().Sdulogin.CasKey)
}

func selfInit() {
	u := &ModuleUser{}
	u.Init()
}
