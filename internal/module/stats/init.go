package stats

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleStats struct{}

func (*ModuleStats) GetName() string {
	return "Stats"
}

func (*ModuleStats) Init() {
	log = logger.New("Stats")
}
