package column

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 北京时区
var beijingLocation = time.FixedZone("CST", 8*60*60)

// getTodayStart 获取北京时间今日零点
func getTodayStart() time.Time {
	now := time.Now().In(beijingLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, beijingLocation)
}

type Column struct {
	Name        string `json:"name" binding:"required,max=75"` // 栏目名称
	Description string `json:"description" binding:"max=200"`  // 栏目描述
	OwnerID     string `json:"owner_id" binding:"required"`    // 栏目创建人学号
	ProjectID   uint   `json:"project_id" binding:"required"`  // 关联的项目ID
	StartDate   int64  `json:"start_date" binding:"required"`  // 栏目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`    // 栏目结束日期
	Avatar      string `json:"avatar"`                         // 栏目封面URL
}

// ColumnCreateReq 定义创建栏目请求的结构体
type ColumnCreateReq struct {
	Name            string `json:"name" binding:"required,max=75"` // 栏目名称
	Description     string `json:"description" binding:"max=200"`  // 栏目描述
	ProjectID       uint   `json:"project_id" binding:"required"`  // 关联的项目ID
	StartDate       int64  `json:"start_date" binding:"required"`  // 栏目开始日期
	EndDate         int64  `json:"end_date" binding:"required"`    // 栏目结束日期
	Avatar          string `json:"avatar"`                         // 栏目封面URL
	DailyPunchLimit int    `json:"daily_punch_limit"`              // 每日可打卡次数，0表示不限次数
	PointEarned     int    `json:"point_earned"`                   // 每次打卡可获得的积分
	StartTime       string `json:"start_time"`                     // 每日打卡开始时间，格式为 "HH:MM"
	EndTime         string `json:"end_time"`                       // 每日打卡结束时间，格式为 "HH:MM"
	Optional        bool   `json:"optional"`                       // 特殊栏目，不计入完成所有栏目的判断
}

// ColumnUpdateReq 定义更新栏目请求的结构体，使用指针类型支持部分更新
type ColumnUpdateReq struct {
	Name            *string `json:"name" binding:"omitempty,max=75"`         // 栏目名称，可选
	Description     *string `json:"description" binding:"omitempty,max=200"` // 栏目描述，可选
	ProjectID       *uint   `json:"project_id"`                              // 关联的项目ID，可选
	StartDate       *int64  `json:"start_date"`                              // 栏目开始日期，可选
	EndDate         *int64  `json:"end_date"`                                // 栏目结束日期，可选
	Avatar          *string `json:"avatar"`                                  // 栏目封面URL，可选
	DailyPunchLimit *int    `json:"daily_punch_limit"`                       // 每日可打卡次数，0表示不限次数
	PointEarned     *int    `json:"point_earned"`                            // 每次打卡可获得的积分
	StartTime       *string `json:"start_time"`                              // 每日打卡开始时间，格式为 "HH:MM"
	EndTime         *string `json:"end_time"`                                // 每日打卡结束时间，格式为 "HH:MM"
	Optional        *bool   `json:"optional"`                                // 特殊栏目，不计入完成所有栏目的判断
}

// ColumnResponse 定义栏目响应结构体（不包含空的Project字段）
type ColumnResponse struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	OwnerID         string `json:"owner_id"`
	ProjectID       uint   `json:"project_id"`
	StartDate       int64  `json:"start_date"`
	EndDate         int64  `json:"end_date"`
	Avatar          string `json:"avatar"`
	DailyPunchLimit int    `json:"daily_punch_limit"`
	PointEarned     uint   `json:"point_earned"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	Optional        bool   `json:"optional"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

// CreateColumn 处理创建栏目请求
func CreateColumn(c *gin.Context) {
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
	var req ColumnCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定创建栏目请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 检查对应的项目是否存在且未被删除
	var project model.Project
	if err := database.DB.First(&project, "id = ?", req.ProjectID).Error; err != nil {
		log.Warn("关联的项目不存在或已被删除", "project_id", req.ProjectID)
		response.Fail(c, response.ErrNotFound.WithTips("关联的项目不存在或已被删除"))
		return
	}

	// 验证栏目的时间范围是否在项目时间范围内
	if req.StartDate < project.StartDate || req.EndDate > project.EndDate {
		log.Warn("栏目时间范围超出项目范围",
			"column_start", req.StartDate, "column_end", req.EndDate,
			"project_start", project.StartDate, "project_end", project.EndDate)
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目的开始和结束时间必须在项目时间范围内"))
		return
	}

	// 验证栏目的开始日期不能晚于结束日期（允许同一天）
	if req.StartDate > req.EndDate {
		log.Warn("栏目开始日期不能晚于结束日期", "start_date", req.StartDate, "end_date", req.EndDate)
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目开始日期不能晚于结束日期"))
		return
	}

	// 验证每日打卡时间设置
	if req.StartTime != "" && req.EndTime != "" {
		startTime, err1 := time.Parse("15:04", req.StartTime)
		endTime, err2 := time.Parse("15:04", req.EndTime)
		if err1 != nil || err2 != nil {
			log.Warn("每日打卡时间格式错误", "start_time", req.StartTime, "end_time", req.EndTime)
			response.Fail(c, response.ErrInvalidRequest.WithTips("每日打卡时间格式错误，应为 HH:MM"))
			return
		}
		// 如果是同一天，开始时间必须早于结束时间（不支持跨天）
		if req.StartDate == req.EndDate && !startTime.Before(endTime) {
			log.Warn("同一天时开始时间必须早于结束时间", "start_time", req.StartTime, "end_time", req.EndTime)
			response.Fail(c, response.ErrInvalidRequest.WithTips("同一天时每日打卡开始时间必须早于结束时间"))
			return
		}
	} else if (req.StartTime != "" && req.EndTime == "") || (req.StartTime == "" && req.EndTime != "") {
		log.Warn("每日打卡时间设置不完整", "start_time", req.StartTime, "end_time", req.EndTime)
		response.Fail(c, response.ErrInvalidRequest.WithTips("每日打卡开始时间和结束时间必须同时设置或同时留空"))
		return
	}

	// 验证积分是否为负数
	if req.PointEarned <= 0 {
		log.Warn("积分必须大于0!", "point_earned", req.PointEarned)
		response.Fail(c, response.ErrInvalidRequest.WithTips("积分必须大于0!"))
		return
	}
	// 创建新的栏目模型
	column := model.Column{
		Name:            req.Name,
		Description:     req.Description,
		OwnerID:         StudentID,
		ProjectID:       req.ProjectID,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Avatar:          req.Avatar,
		DailyPunchLimit: req.DailyPunchLimit,
		PointEarned:     uint(req.PointEarned),
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		Optional:        req.Optional,
	}

	if err := database.DB.Create(&column).Error; err != nil {
		log.Error("创建栏目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	response.Success(c, gin.H{"column_id": column.ID})
}

// UpdateColumn 处理更新栏目请求
func UpdateColumn(c *gin.Context) {
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
	var req ColumnUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定更新栏目请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	var column model.Column
	if err := database.DB.Where("id = ? AND owner_id = ?", c.Param("id"), StudentID).First(&column).Error; err != nil {
		log.Error("查询栏目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在或无权限"))
		return
	}

	// 如果要更新ProjectID或时间字段，需要验证
	if req.ProjectID != nil || req.StartDate != nil || req.EndDate != nil {
		// 获取项目信息（如果更新了ProjectID，使用新的；否则使用原有的）
		projectID := column.ProjectID
		if req.ProjectID != nil {
			projectID = *req.ProjectID
		}

		var project model.Project
		if err := database.DB.First(&project, "id = ?", projectID).Error; err != nil {
			log.Warn("关联的项目不存在或已被删除", "project_id", projectID)
			response.Fail(c, response.ErrNotFound.WithTips("关联的项目不存在或已被删除"))
			return
		}

		// 计算更新后的时间范围
		startDate := column.StartDate
		endDate := column.EndDate
		if req.StartDate != nil {
			startDate = *req.StartDate
		}
		if req.EndDate != nil {
			endDate = *req.EndDate
		}

		// 验证栏目的时间范围是否在项目时间范围内
		if startDate < project.StartDate || endDate > project.EndDate {
			log.Warn("栏目时间范围超出项目范围",
				"column_start", startDate, "column_end", endDate,
				"project_start", project.StartDate, "project_end", project.EndDate)
			response.Fail(c, response.ErrInvalidRequest.WithTips("栏目的开始和结束时间必须在项目时间范围内"))
			return
		}

		// 验证栏目的开始日期不能晚于结束日期（允许同一天）
		if startDate > endDate {
			log.Warn("栏目开始日期不能晚于结束日期", "start_date", startDate, "end_date", endDate)
			response.Fail(c, response.ErrInvalidRequest.WithTips("栏目开始日期不能晚于结束日期"))
			return
		}

	}

	// 验证每日打卡时间设置（更新时需要考虑原有值）
	startTime := column.StartTime
	endTime := column.EndTime
	if req.StartTime != nil {
		startTime = *req.StartTime
	}
	if req.EndTime != nil {
		endTime = *req.EndTime
	}
	// 计算更新后的日期
	finalStartDate := column.StartDate
	finalEndDate := column.EndDate
	if req.StartDate != nil {
		finalStartDate = *req.StartDate
	}
	if req.EndDate != nil {
		finalEndDate = *req.EndDate
	}

	if startTime != "" && endTime != "" {
		parsedStartTime, err1 := time.Parse("15:04", startTime)
		parsedEndTime, err2 := time.Parse("15:04", endTime)
		if err1 != nil || err2 != nil {
			log.Warn("每日打卡时间格式错误", "start_time", startTime, "end_time", endTime)
			response.Fail(c, response.ErrInvalidRequest.WithTips("每日打卡时间格式错误，应为 HH:MM"))
			return
		}
		// 如果是同一天，开始时间必须早于结束时间（不支持跨天）
		if finalStartDate == finalEndDate && !parsedStartTime.Before(parsedEndTime) {
			log.Warn("同一天时开始时间必须早于结束时间", "start_time", startTime, "end_time", endTime)
			response.Fail(c, response.ErrInvalidRequest.WithTips("同一天时每日打卡开始时间必须早于结束时间"))
			return
		}
	} else if (startTime != "" && endTime == "") || (startTime == "" && endTime != "") {
		log.Warn("每日打卡时间设置不完整", "start_time", startTime, "end_time", endTime)
		response.Fail(c, response.ErrInvalidRequest.WithTips("每日打卡开始时间和结束时间必须同时设置或同时留空"))
		return
	}

	if req.PointEarned != nil {
		// 验证积分是否为负数
		if *req.PointEarned <= 0 {
			log.Warn("积分必须大于0!", "point_earned", req.PointEarned)
			response.Fail(c, response.ErrInvalidRequest.WithTips("积分必须大于0!"))
		}
		column.PointEarned = uint(*req.PointEarned)
	}
	if req.Name != nil {
		column.Name = *req.Name
	}
	if req.Description != nil {
		column.Description = *req.Description
	}
	if req.ProjectID != nil {
		column.ProjectID = *req.ProjectID
	}
	if req.StartDate != nil {
		column.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		column.EndDate = *req.EndDate
	}
	if req.Avatar != nil {
		column.Avatar = *req.Avatar
	}
	if req.DailyPunchLimit != nil {
		column.DailyPunchLimit = *req.DailyPunchLimit
	}

	if req.StartTime != nil {
		column.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		column.EndTime = *req.EndTime
	}
	if req.Optional != nil {
		column.Optional = *req.Optional
	}

	if err := database.DB.Save(&column).Error; err != nil {
		log.Error("更新栏目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	response.Success(c, gin.H{"column_id": column.ID})
}

// DeleteColumn 处理删除栏目请求
func DeleteColumn(c *gin.Context) {
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

	// 获取栏目ID
	id := c.Param("id")
	if id == "" {
		log.Error("栏目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目ID不能为空"))
		return
	}

	// 查询栏目是否存在
	var column model.Column
	if err := database.DB.First(&column, "id = ? AND owner_id = ?", id, StudentID).Error; err != nil {
		log.Error("查询栏目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在或无权限"))
		return
	}

	if err := database.DB.Delete(&column).Error; err != nil {
		log.Error("删除栏目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	response.Success(c)
}

// GetColumn 处理获取栏目详情请求
func GetColumn(c *gin.Context) {
	// 获取栏目ID
	id := c.Param("id")
	if id == "" {
		log.Error("栏目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目ID不能为空"))
		return
	}

	// 获取认证信息
	payload, exists := c.Get("payload")
	var userID uint
	if exists {
		userPayload, ok := payload.(*jwt.Claims)
		if ok {
			userID = userPayload.ID
		}
	}

	var column model.Column
	// 查询栏目详情，确保关联的项目和活动都未被删除
	if err := database.DB.Joins("JOIN project ON project.id = column.project_id AND project.deleted_at IS NULL").
		Joins("JOIN activity ON activity.id = project.activity_id AND activity.deleted_at IS NULL").
		Preload("Project").Preload("User").
		First(&column, "column.id = ?", id).Error; err != nil {
		log.Error("查询栏目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("栏目被删除或不存在"))
		return
	}

	// 构建响应数据
	responseData := gin.H{
		"id":                column.ID,
		"name":              column.Name,
		"description":       column.Description,
		"owner_id":          column.OwnerID,
		"project_id":        column.ProjectID,
		"start_date":        column.StartDate,
		"end_date":          column.EndDate,
		"avatar":            column.Avatar,
		"daily_punch_limit": column.DailyPunchLimit,
		"point_earned":      column.PointEarned,
		"start_time":        column.StartTime,
		"end_time":          column.EndTime,
		"optional":          column.Optional,
		"created_at":        column.CreatedAt.Unix(),
		"updated_at":        column.UpdatedAt.Unix(),
		"project":           column.Project,
		"user":              column.User,
		"punched_today":     false,
		"today_punch_count": 0,
	}

	// 如果用户已登录，查询今日打卡状态（使用北京时间）
	if userID > 0 {
		today := getTodayStart()
		var todayPunchCount int64
		database.DB.Model(&model.Punch{}).Where("column_id = ? AND user_id = ? AND created_at >= ?", id, userID, today).Count(&todayPunchCount)

		responseData["punched_today"] = todayPunchCount == int64(column.DailyPunchLimit)
		responseData["today_punch_count"] = todayPunchCount
	}

	response.Success(c, responseData)
}

// ListColumns 处理获取栏目列表请求
func ListColumns(c *gin.Context) {

	var columns []model.Column
	// 查询栏目，确保关联的项目和活动未被删除
	if err := database.DB.Joins("JOIN project ON project.id = column.project_id AND project.deleted_at IS NULL").
		Joins("JOIN activity ON activity.id = project.activity_id AND activity.deleted_at IS NULL").
		Preload("Project").Preload("User").
		Find(&columns).Error; err != nil {
		log.Error("查询栏目列表失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	var projectResponses []ColumnResponse
	for _, p := range columns {
		projectResponses = append(projectResponses, ColumnResponse{
			ID:              p.ID,
			Name:            p.Name,
			Description:     p.Description,
			OwnerID:         p.OwnerID,
			ProjectID:       p.ProjectID,
			StartDate:       p.StartDate,
			EndDate:         p.EndDate,
			Avatar:          p.Avatar,
			DailyPunchLimit: p.DailyPunchLimit,
			PointEarned:     p.PointEarned,
			StartTime:       p.StartTime,
			EndTime:         p.EndTime,
			Optional:        p.Optional,
			CreatedAt:       p.CreatedAt.Unix(),
			UpdatedAt:       p.UpdatedAt.Unix(),
		})
	}

	response.Success(c, projectResponses)
}

// RestoreColumn 处理恢复已删除栏目的请求
func RestoreColumn(c *gin.Context) {
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

	// 获取栏目ID
	id := c.Param("id")
	if id == "" {
		log.Error("栏目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目ID不能为空"))
		return
	}

	var column model.Column
	if err := database.DB.Unscoped().Where("id = ? AND owner_id = ?", id, StudentID).First(&column).Error; err != nil {
		log.Error("查询栏目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在或无权限"))
		return
	}

	column.DeletedAt = gorm.DeletedAt{}
	if err := database.DB.Unscoped().Save(&column).Error; err != nil {
		log.Error("恢复栏目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	response.Success(c, gin.H{"column_id": column.ID})
}
