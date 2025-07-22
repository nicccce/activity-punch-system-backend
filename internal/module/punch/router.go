package punch

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (p *ModulePunch) InitRouter(r *gin.RouterGroup) {
	// 定义打卡模块的路由组，所有打卡相关端点以 /punch 为前缀
	punchGroup := r.Group("/punch")

	punchGroup.Use(middleware.Auth(1))
	{
		// 注册审核打卡记录端点
		punchGroup.POST("/review", ReviewPunch)
	}

	punchGroup.Use(middleware.Auth(0))
	{

		// 注册插入打卡记录端点
		punchGroup.POST("/insert", InsertPunch)
		// 注册获取打卡记录端点
		punchGroup.GET("/:column_id", GetPunchesByColumn)
		// 其他打卡相关路由可在此注册
	}
}
