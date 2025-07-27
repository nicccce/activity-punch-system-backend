package project

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (p *ModuleProject) InitRouter(r *gin.RouterGroup) {
	// 定义项目模块的路由组，所有项目相关端点以 /project 为前缀
	activityGroup := r.Group("/project")
	adminGroup := r.Group("/project")

	activityGroup.Use(middleware.Auth(0))
	{
		// 注册获取项目列表端点
		activityGroup.GET("/list", ListProjects)

		// 注册获取单个项目端点
		activityGroup.GET("/get/:id", GetProject)
	}

	adminGroup.Use(middleware.Auth(1))
	{
		// 注册创建项目端点
		adminGroup.POST("/create", CreateProject)

		// 注册更新项目端点
		adminGroup.PUT("/update/:id", UpdateProject)

		// 注册删除项目端点
		adminGroup.DELETE("/delete/:id", DeleteProject)

		// 还原删除项目端点
		adminGroup.PUT("/restore/:id", RestoreProject)
	}
}
