package column

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

func (c *ModuleColumn) InitRouter(r *gin.RouterGroup) {
	// 定义列模块的路由组，所有列相关端点以 /column 为前缀
	activityGroup := r.Group("/column")

	activityGroup.Use(middleware.Auth(0))
	{
		// 注册获取列列表端点
		activityGroup.GET("/list", ListColumns)

		// 注册获取单个列端点
		activityGroup.GET("/get/:id", GetColumn)
	}
	activityGroup.Use(middleware.Auth(1))
	{
		// 注册创建列端点
		activityGroup.POST("/create", CreateColumn)

		// 注册更新列端点
		activityGroup.PUT("/update/:id", UpdateColumn)

		// 注册删除列端点
		activityGroup.DELETE("/delete/:id", DeleteColumn)

		// 还原删除列端点
		activityGroup.PUT("/restore/:id", RestoreColumn)
	}
}
