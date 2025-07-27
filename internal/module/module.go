package module

import (
	"activity-punch-system/internal/module/activity"
	"activity-punch-system/internal/module/column"
	"activity-punch-system/internal/module/ping"
	"activity-punch-system/internal/module/project"
	"activity-punch-system/internal/module/punch"
	"activity-punch-system/internal/module/star"
	"activity-punch-system/internal/module/user"

	"github.com/gin-gonic/gin"
)

type Module interface {
	GetName() string
	Init()
	InitRouter(r *gin.RouterGroup)
}

var Modules []Module

func registerModule(m []Module) {
	Modules = append(Modules, m...)
}

func init() {
	// Register your module here
	registerModule([]Module{
		&user.ModuleUser{},
		&ping.ModulePing{},
		&activity.ModuleActivity{},
		&project.ModuleProject{},
		&column.ModuleColumn{},
		&punch.ModulePunch{},
		&star.ModuleStar{},
	})
}
