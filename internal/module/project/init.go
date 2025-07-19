package project

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleProject struct{}

func (p *ModuleProject) GetName() string {
	return "Project"
}

func (u *ModuleProject) Init() {
	log = logger.New("Project")
}

func selfInit() {
	u := &ModuleProject{}
	u.Init()
}
