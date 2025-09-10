package column

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/logger"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"activity-punch-system/tools"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"math"
)

// Record todo: 此仅作为占位
type Record struct{}

var log = logger.New("Stats-Column")

// Brief 获取某栏目的今日(若给定时间则为给定时间当天最早时间之后的)已经打卡人数,
// 请求者参与打卡次数,
// 请求者在该栏目下的总得分,
// 以及请求者在该栏目下的排名(按打卡总得分排名)
// todo: 这几者揉在一起的合理性有待考量
func Brief(c *gin.Context) {
	user, ok := jwt.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	columnId, ok := columnIdValidator(c)
	if !ok {
		return
	}
	var result briefResult
	askTime := tools.GetTime(c)
	if err := briefStats(columnId, user.Id, askTime, &result); err != nil {
		log.Error("查询 column 表错误", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	response.Success(c, result)
}

// Rank 获取某栏目的今日的(给定时间之前的)打卡人员排名
// rankByCount: bool,控制是否按打卡次数排名,默认false
func Rank(c *gin.Context) {
	columnId, ok := columnIdValidator(c)
	if !ok {
		return
	}
	askTime := tools.GetTime(c)
	offset, limit := tools.GetPage(c)
	var result []rankResult
	var err error
	if rbc := c.Query("rankByCount"); rbc == "true" {
		err = rankByCount(columnId, askTime, offset, limit, &result)
	} else {
		err = rankByScore(columnId, askTime, offset, limit, &result)
	}
	if err != nil {
		log.Error("查询 column 表错误", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	response.Success(c, result)
}

// Recent 获取请求用户的最近打卡记录(不限colum)
//func Recent(c *gin.Context) {
//	user, ok := context.GetUserPayload(c)
//	if !ok {
//		response.Fail(c, response.ErrUnauthorized)
//		return
//	}
//	var result []Record
//	offset, limit := tool.GetPage(c)
//	askTime := tool.GetTime(c)
//	err := selectRecordsByStudentId(user.Id, askTime, offset, limit, &result)
//	if err != nil {
//		log.Error("查询 column 表错误", "error", err)
//		response.Fail(c, response.ErrDatabase.WithOrigin(err))
//		return
//	}
//}

func Export2Json(whereUser bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		columnId, ok := columnIdValidator(c)
		if !ok {
			return
		}
		var records []Record
		offset, limit := tools.GetPage(c)
		askTime := tools.GetTime(c)
		err := selectRecords(columnId, askTime, offset, limit, &records, "")
		if err != nil {
			log.Error("查询 column 表错误", "error", err)
			response.Fail(c, response.ErrDatabase.WithOrigin(err))
			return
		}
		response.Success(c, records)
	}
}

// Export2Excel todo: 也许可改为异步订阅任务模式
func Export2Excel() gin.HandlerFunc {
	return func(c *gin.Context) {
		columnId, ok := columnIdValidator(c)
		if !ok {
			return
		}
		var records []Record
		askTime := tools.GetTime(c)
		err := selectRecords(columnId, askTime, 0, math.MaxInt, &records, "")
		if err != nil {
			log.Error("查询 column 表错误", "error", err)
			response.Fail(c, response.ErrDatabase.WithOrigin(err))
			return
		}
		f := excelize.NewFile()
		defer tools.PanicOnErr(f.Close())
		err = tools.ExportToExcel(f, "", records)
		if err != nil {
			log.Error("导出 excel 错误", "error", err)
			response.Fail(c, response.ErrDatabase.WithOrigin(err))
			return
		}
		c.Header("Content-Type", "application/octet-stream")
		filename := fmt.Sprintf("%s_%d.xlsx", columnId, askTime)

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Cache-Control", "must-revalidate")
		c.Header("Pragma", "public")
		c.Header("Expires", "0")
		_ = f.Write(c.Writer)
	}
}

func columnIdValidator(c *gin.Context) (string, bool) {
	columnId := c.Param("id")
	if columnId == "" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目ID不能为空"))
		return "", false
	}
	// 是否需要确保尚未删除?
	{
		var count int64
		r := database.DB.Model(&model.Column{}).
			Joins("JOIN project ON project.id = column.project_id AND project.deleted_at IS NULL").
			Joins("JOIN activity ON activity.id = project.activity_id AND activity.deleted_at IS NULL").
			Where("column.id = ?", columnId).
			Count(&count)

		if r.Error != nil {
			log.Error("查询 column 表错误", "error", r.Error)
			response.Fail(c, response.ErrDatabase.WithOrigin(r.Error))
			return "", false
		} else if count == 0 {
			response.Fail(c, response.ErrNotFound)
			return "", false
		} else if count > 1 {
			log.Warn("查询 column 表警告", "error", "重复 columnId")
		}
	}
	return columnId, true
}
