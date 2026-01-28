package stats

import (
	"activity-punch-system/internal/global/logger"
	"activity-punch-system/internal/module/stats/activity"
	"activity-punch-system/internal/module/stats/column"
	"log/slog"
)

var log *slog.Logger

type ModuleStats struct{}

func (*ModuleStats) GetName() string {
	return "Stats"
}

func (*ModuleStats) Init() {
	log = logger.New("Stats")
	column.Log = log
	activity.Log = log
}
