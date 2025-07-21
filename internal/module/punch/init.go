package punch

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleActivity struct{}

func (p *ModuleActivity) GetName() string {
	return "Punch"
}

func (u *ModuleActivity) Init() {
	log = logger.New("Punch")
}

func selfInit() {
	u := &ModuleActivity{}
	u.Init()
}
