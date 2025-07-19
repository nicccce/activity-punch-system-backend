package project

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (p *ModuleProject) InitRouter(r *gin.RouterGroup) {
	// 定义项目模块的路由组，所有项目相关端点以 /project 为前缀
	projectGroup := r.Group("/project")

	projectGroup.Use(middleware.Auth(0))
	{
		// 注册获取项目列表端点
		projectGroup.GET("/list", ListProjects)
	}

	projectGroup.Use(middleware.Auth(1))
	{
		// 注册创建项目端点
		projectGroup.POST("/create", CreateProject)

		// 注册更新项目端点
		projectGroup.PUT("/update/:id", UpdateProject)

		// 注册删除项目端点
		projectGroup.DELETE("/delete/:id", DeleteProject)

		// 还原删除项目端点
		projectGroup.PUT("/restore/:id", RestoreProject)
	}
}
