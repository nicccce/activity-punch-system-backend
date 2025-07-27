// Package star 根据需求文档，仅支持 管理员 对 打卡
package star

import (
	"activity-punch-system/internal/global/middleware"
	"github.com/gin-gonic/gin"
)

func (*ModuleStar) InitRouter(r *gin.RouterGroup) {

	starGroup := r.Group("/star")

	commonGroup := starGroup.Use(middleware.Auth(0))
	{
		commonGroup.GET("/count", count)
	}

	adminGroup := starGroup.Use(middleware.Auth(1))
	{
		adminGroup.POST("/add", add)
		adminGroup.GET("/list", list)
		adminGroup.DELETE("/cancel", cancel)
		adminGroup.GET("/ask", ask)
	}
}
