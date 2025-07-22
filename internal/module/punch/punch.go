package punch

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/pictureBed"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"mime/multipart"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PunchInsertRequest 定义插入打卡记录的请求体结构
// @Description 插入打卡记录请求体
// @Param column_id int 打卡栏目ID
// @Param content string 打卡内容
// @Param images []*multipart.FileHeader 图片数组
// @example multipart form-data: column_id=1, content=xxx, images=[file1, file2]
type PunchInsertRequest struct {
	ColumnID int                     `form:"column_id" binding:"required"`
	Content  string                  `form:"content" binding:"required"`
	Images   []*multipart.FileHeader `form:"images" binding:"omitempty"`
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

	// 绑定 multipart form-data
	var req PunchInsertRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Error("绑定打卡请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 获取栏目时间范围，判断是否允许打卡
	var column model.Column
	if err := database.DB.First(&column, "id = ?", req.ColumnID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在"))
		return
	}
	startDateStr := strconv.FormatInt(column.StartDate, 10)
	endDateStr := strconv.FormatInt(column.EndDate, 10)
	startDate, err := time.Parse("20060102", startDateStr)
	if err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目开始时间格式错误"))
		return
	}
	endDate, err := time.Parse("20060102", endDateStr)
	if err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目结束时间格式错误"))
		return
	}
	currentTime := time.Now()
	if currentTime.Before(startDate) || currentTime.After(endDate) {
		response.Fail(c, response.ErrInvalidRequest.WithTips("当前时间不在栏目时间范围内，无法打卡"))
		return
	}

	punch := &model.Punch{
		ColumnID: req.ColumnID,
		UserID:   StudentID,
		Content:  req.Content,
		Status:   0, // 默认待审核
	}

	if err := database.DB.Create(punch).Error; err != nil {
		log.Error("插入打卡记录失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 处理图片保存和punch_img插入
	if len(req.Images) > 0 {
		pictureBed := pictureBed.NewPictureBed("./upload/punch", "/static/punch")
		for _, fileHeader := range req.Images {
			imgUrl, err := pictureBed.SaveImage(fileHeader)
			if err != nil {
				log.Error("图片保存失败", "error", err)
				continue
			}
			punchImg := &model.PunchImg{
				PunchID:  punch.ID, // 新增 punch_id 字段
				ColumnID: req.ColumnID,
				ImgURL:   imgUrl,
			}
			database.DB.Create(punchImg)
		}
	}

	response.Success(c, punch)
}

type ReviewReq struct {
	PunchID int `json:"punch_id" binding:"required"`
	Status  int `json:"status" binding:"required"` // 1: 通过, 2: 拒绝
}

// ReviewPunch 审核打卡记录
func ReviewPunch(c *gin.Context) {
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

	// 只允许管理员或有权限的用户审核
	if userPayload.RoleID < 1 { // 假设1为审核权限
		response.Fail(c, response.ErrForbidden)
		return
	}

	// 获取打卡ID和审核状态
	var req ReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定审核请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 查找打卡记录
	var punch model.Punch
	if err := database.DB.First(&punch, req.PunchID).Error; err != nil {
		log.Warn("打卡记录不存在", "punch_id", req.PunchID)
		response.Fail(c, response.ErrNotFound.WithTips("打卡记录不存在"))
		return
	}

	// 更新审核状态
	punch.Status = req.Status
	if err := database.DB.Save(&punch).Error; err != nil {
		log.Error("审核打卡记录失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	response.Success(c, punch)
}

// GetPunchesByColumn 查询某栏目下所有打卡记录
func GetPunchesByColumn(c *gin.Context) {
	columnIDStr := c.Param("column_id")
	if columnIDStr == "" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目ID不能为空"))
		return
	}
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
	studentID := userPayload.StudentID

	var punches []model.Punch
	// 查询当前用户未被删除的打卡记录
	if err := database.DB.Where("column_id = ? AND user_id = ? AND deleted_at IS NULL", columnIDStr, studentID).Find(&punches).Error; err != nil {
		log.Error("查询打卡记录失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 查询每条打卡记录的图片（左连接）
	type PunchWithImg struct {
		model.Punch
		ImgURL string `json:"img_url"`
	}
	var result []PunchWithImg
	for _, punch := range punches {
		var img model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).First(&img)
		result = append(result, PunchWithImg{
			Punch:  punch,
			ImgURL: img.ImgURL, // 没有图片则为空
		})
	}

	// 查询该栏目下未被删除的不同 user_id 数量
	var userCount int64
	database.DB.Model(&model.Punch{}).Where("column_id = ? AND deleted_at IS NULL", columnIDStr).Distinct("user_id").Count(&userCount)

	// 查询当前用户未被删除的打卡数量
	var myCount int64
	database.DB.Model(&model.Punch{}).Where("column_id = ? AND user_id = ? AND deleted_at IS NULL", columnIDStr, studentID).Count(&myCount)

	response.Success(c, gin.H{
		"records":    result,
		"user_count": userCount,
		"my_count":   myCount,
	})
}

// DeletePunch 删除自己拥有的打卡记录（只能删除自己的且时间在栏目范围内）
func DeletePunch(c *gin.Context) {
	punchID := c.Param("id")
	if punchID == "" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("打卡ID不能为空"))
		return
	}
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
	studentID := userPayload.StudentID

	var punch model.Punch
	if err := database.DB.First(&punch, "id = ? AND user_id = ?", punchID, studentID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("打卡记录不存在或无权限"))
		return
	}

	var column model.Column
	if err := database.DB.First(&column, "id = ?", punch.ColumnID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在"))
		return
	}

	// 判断打卡时间是否在栏目时间范围内（startdate/enddate为int64，需转为string再转为date）
	startDateStr := strconv.FormatInt(column.StartDate, 10)
	endDateStr := strconv.FormatInt(column.EndDate, 10)
	startDate, err := time.Parse("20060102", startDateStr)
	if err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目开始时间格式错误"))
		return
	}
	endDate, err := time.Parse("20060102", endDateStr)
	if err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目结束时间格式错误"))
		return
	}
	if punch.CreatedAt.Before(startDate) || punch.CreatedAt.After(endDate) {
		response.Fail(c, response.ErrInvalidRequest.WithTips("打卡时间不在栏目时间范围内，无法删除"))
		return
	}

	if err := database.DB.Delete(&punch).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	response.Success(c, gin.H{"deleted": true})
}
