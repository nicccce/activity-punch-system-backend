package user

import (
	"activity-punch-system-backend/internal/global/logger"
	"activity-punch-system-backend/test"
	"log/slog"
)

var log *slog.Logger

type ModuleUser struct{}

func (u *ModuleUser) GetName() string {
	return "User"
}

func (u *ModuleUser) Init() {
	switch test.IsTest() {
	case false:
		log = logger.New("User")
	case true:
		log = logger.Get()
	}
}

func selfInit() {
	u := &ModuleUser{}
	u.Init()
}
