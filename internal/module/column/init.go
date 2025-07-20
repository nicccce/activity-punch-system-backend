package column

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleColumn struct{}

func (c *ModuleColumn) GetName() string {
	return "Column"
}
func (c *ModuleColumn) Init() {
	log = logger.New("Column")
}
func selfInit() {
	c := &ModuleColumn{}
	c.Init()
}
