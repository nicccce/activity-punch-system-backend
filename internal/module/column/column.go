package column

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Column struct {
	Name        string `json:"name" binding:"required"`       // 栏目名称
	Description string `json:"description"`                   // 栏目描述
	OwnerID     string `json:"owner_id" binding:"required"`   // 栏目创建人学号
	ProjectID   uint   `json:"project_id" binding:"required"` // 关联的项目ID
	StartDate   int64  `json:"start_date" binding:"required"` // 栏目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`   // 栏目结束日期
	Avatar      string `json:"avatar"`                        // 栏目封面URL
}

// ColumnCreateReq 定义创建栏目请求的结构体
type ColumnCreateReq struct {
	Name        string `json:"name" binding:"required"`       // 栏目名称
	Description string `json:"description"`                   // 栏目描述
	ProjectID   uint   `json:"project_id" binding:"required"` // 关联的项目ID
	StartDate   int64  `json:"start_date" binding:"required"` // 栏目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`   // 栏目结束日期
	Avatar      string `json:"avatar"`                        // 栏目封面URL
}

// ColumnUpdateReq 定义更新栏目请求的结构体，使用指针类型支持部分更新
type ColumnUpdateReq struct {
	Name        *string `json:"name"`        // 栏目名称，可选
	Description *string `json:"description"` // 栏目描述，可选
	ProjectID   *uint   `json:"project_id"`  // 关联的项目ID，可选
	StartDate   *int64  `json:"start_date"`  // 栏目开始日期，可选
	EndDate     *int64  `json:"end_date"`    // 栏目结束日期，可选
	Avatar      *string `json:"avatar"`      // 栏目封面URL，可选
}

// ColumnResponse 定义栏目响应结构体（不包含空的Project字段）
type ColumnResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	ProjectID   uint   `json:"project_id"`
	StartDate   int64  `json:"start_date"`
	EndDate     int64  `json:"end_date"`
	Avatar      string `json:"avatar"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
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

	// 验证栏目的开始时间不能晚于结束时间
	if req.StartDate >= req.EndDate {
		log.Warn("栏目开始时间不能晚于或等于结束时间", "start_date", req.StartDate, "end_date", req.EndDate)
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目开始时间必须早于结束时间"))
		return
	}

	// 创建新的栏目模型
	column := model.Column{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     StudentID,
		ProjectID:   req.ProjectID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Avatar:      req.Avatar,
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

		// 验证栏目的开始时间不能晚于结束时间
		if startDate >= endDate {
			log.Warn("栏目开始时间不能晚于或等于结束时间", "start_date", startDate, "end_date", endDate)
			response.Fail(c, response.ErrInvalidRequest.WithTips("栏目开始时间必须早于结束时间"))
			return
		}
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

	response.Success(c, column)
}

// ListColumns 处理获取栏目列表请求
func ListColumns(c *gin.Context) {
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

	var columns []model.Column
	// 查询栏目，确保关联的项目和活动未被删除
	if err := database.DB.Joins("JOIN project ON project.id = column.project_id AND project.deleted_at IS NULL").
		Joins("JOIN activity ON activity.id = project.activity_id AND activity.deleted_at IS NULL").
		Where("column.owner_id = ?", StudentID).
		Preload("Project").Preload("User").
		Find(&columns).Error; err != nil {
		log.Error("查询栏目列表失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	var projectResponses []ColumnResponse
	for _, p := range columns {
		projectResponses = append(projectResponses, ColumnResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			OwnerID:     p.OwnerID,
			ProjectID:   p.ProjectID,
			StartDate:   p.StartDate,
			EndDate:     p.EndDate,
			Avatar:      p.Avatar,
			CreatedAt:   p.CreatedAt.Unix(),
			UpdatedAt:   p.UpdatedAt.Unix(),
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
