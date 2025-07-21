package punch

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// PunchInsertRequest 定义插入打卡记录的请求体结构
// @Description 插入打卡记录请求体
// @Param column_id int 打卡栏目ID
// @Param content string 打卡内容
// @example {"column_id":1, "content":"今日打卡内容"}
type PunchInsertRequest struct {
	ColumnID int    `json:"column_id" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

// InsertPunch 插入一条打卡记录
func InsertPunch(c *gin.Context) {

	// 获取认证信息
	payload, exists := c.Get("payload")
	if !exists {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	userPayload, ok := payload.(*jwt.Claims)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	StudentID := userPayload.StudentID

	// 定义请求结构体并绑定 JSON 数据
	var req PunchInsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定更新项目请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	punch := &model.Punch{
		ColumnID: req.ColumnID,
		UserID:   StudentID,
		Content:  req.Content,
		Status:   0, // 默认待审核
	}

}
