package punch

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (p *ModulePunch) InitRouter(r *gin.RouterGroup) {
	// 定义打卡模块的路由组，所有打卡相关端点以 /punch 为前缀
	punchGroup := r.Group("/punch")

	adminGroup := punchGroup.Use(middleware.Auth(1))
	{
		// 审核打卡记录端点
		adminGroup.POST("/review", ReviewPunch)
		adminGroup.GET("/pending-list", GetPendingPunchList)
	}

	commonGroup := punchGroup.Use(middleware.Auth(0))
	{

		// 插入打卡记录端点
		commonGroup.POST("/insert", InsertPunch)
		// 获取打卡记录端点
		commonGroup.GET("/:column_id", GetPunchesByColumn)

		// 删除打卡记录端点
		commonGroup.DELETE("/delete/:id", DeletePunch)
		// 修改打卡记录端点
		commonGroup.PUT("/update/:id", UpdatePunch)
		// 查询自己所有打卡记录端点
		commonGroup.GET("/my-list", GetMyPunchList)
		// 获取最近参与栏目、项目、活动端点
		commonGroup.GET("/recent-participation", GetRecentParticipation)
	}
}
