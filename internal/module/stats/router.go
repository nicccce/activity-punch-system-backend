package stats

import (
	"activity-punch-system/internal/global/middleware"
	"activity-punch-system/internal/module/stats/activity"
	"activity-punch-system/internal/module/stats/column"
	"github.com/gin-gonic/gin"
)

func (*ModuleStats) InitRouter(r *gin.RouterGroup) {
	commonGroup := r.Group("/stats")
	commonGroup.Use(middleware.Auth(0))
	{
		columnCommon := commonGroup.Group("/column/:id")
		{
			//commonGroup.POST("/recent", column.Recent)
			columnCommon.GET("/brief", column.Brief)
			columnCommon.POST("/rank", column.Rank)
			//columnCommon.POST("/record", column.Records)
		}
		activityCommon := commonGroup.Group("/activity")
		{
			activityCommon.POST("/history", activity.History)
			activityCommon.POST("/:id/rank", activity.Rank)
			activityCommon.POST("/:id/detail", activity.Detail)
			activityCommon.GET("/:id/brief", activity.Brief)
			activityCommon.GET("/:id/rank/export", activity.RankExport)
		}
	}
	adminGroup := r.Group("/stats")
	adminGroup.Use(middleware.Auth(1))
	{
		//columnGroup := adminGroup.Group("/column/:id")
		//{
		//columnGroup.POST("/export", column.Export2Json())
		//commonGroup.POST("/export/rank.xlsx", column.Export2Excel)
		//}
		activityAdmin := commonGroup.Group("/activity")
		{
			activityAdmin.GET("/:id/export", activity.Export)
		}
	}
}
