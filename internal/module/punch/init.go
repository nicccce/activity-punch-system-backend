package punch

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModulePunch struct{}

func (p *ModulePunch) GetName() string {
	return "Punch"
}

func (u *ModulePunch) Init() {
	log = logger.New("Punch")
}

func selfInit() {
	u := &ModulePunch{}
	u.Init()
}
