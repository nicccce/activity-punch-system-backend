package user

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleUser struct{}

func (u *ModuleUser) GetName() string {
	return "User"
}

func (u *ModuleUser) Init() {
	log = logger.New("User")
}

func selfInit() {
	u := &ModuleUser{}
	u.Init()
}
