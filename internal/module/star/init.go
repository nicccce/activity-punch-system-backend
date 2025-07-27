package star

import (
	"activity-punch-system/internal/global/logger"
	"log/slog"
)

var log *slog.Logger

type ModuleStar struct{}

func (*ModuleStar) GetName() string {
	return "Star"
}

func (*ModuleStar) Init() {
	log = logger.New("Star")
}
