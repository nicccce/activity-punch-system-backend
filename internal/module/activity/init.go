package activity

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleActivity struct{}

func (p *ModuleActivity) GetName() string {
	return "Activity"
}

func (u *ModuleActivity) Init() {
	log = logger.New("Activity")
}

func selfInit() {
	u := &ModuleActivity{}
	u.Init()
}
