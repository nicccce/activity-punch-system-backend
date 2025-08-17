package column

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (c *ModuleColumn) InitRouter(r *gin.RouterGroup) {
	// 定义栏目模块的路由组，所有栏目相关端点以 /column 为前缀
	activityGroup := r.Group("/column")
	adminGroup := r.Group("/column")

	activityGroup.Use(middleware.Auth(0))
	{
		// 注册获取栏目栏目端点
		activityGroup.GET("/list", ListColumns)

		// 注册获取单个栏目端点
		activityGroup.GET("/get/:id", GetColumn)
	}
	adminGroup.Use(middleware.Auth(1))
	{
		// 注册创建栏目端点
		adminGroup.POST("/create", CreateColumn)

		// 注册更新栏目端点
		adminGroup.PUT("/update/:id", UpdateColumn)

		// 注册删除栏目端点
		adminGroup.DELETE("/delete/:id", DeleteColumn)

		// 还原删除栏目端点
		adminGroup.PUT("/restore/:id", RestoreColumn)
	}
}
