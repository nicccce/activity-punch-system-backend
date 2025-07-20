package activity

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (p *ModuleActivity) InitRouter(r *gin.RouterGroup) {
	// 定义项目模块的路由组，所有项目相关端点以 /activity 为前缀
	activityGroup := r.Group("/activity")

	activityGroup.Use(middleware.Auth(0))
	{
		// 注册获取项目列表端点
		activityGroup.GET("/list", ListActivitys)

		// 注册获取单个项目端点
		activityGroup.GET("/get/:id", GetActivity)
	}

	activityGroup.Use(middleware.Auth(1))
	{
		// 注册创建项目端点
		activityGroup.POST("/create", CreateActivity)

		// 注册更新项目端点
		activityGroup.PUT("/update/:id", UpdateActivity)

		// 注册删除项目端点
		activityGroup.DELETE("/delete/:id", DeleteActivity)

		// 还原删除项目端点
		activityGroup.PUT("/restore/:id", RestoreActivity)
	}
}
