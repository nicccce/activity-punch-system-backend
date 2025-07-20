package project

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Project struct {
	Name        string `json:"name" binding:"required"`        // 项目名称
	Description string `json:"description"`                    // 项目描述
	OwnerID     string `json:"owner_id" binding:"required"`    // 项目创建人学号
	ActivityID  uint   `json:"activity_id" binding:"required"` // 关联的活动ID
	StartDate   int64  `json:"start_date" binding:"required"`  // 项目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`    // 项目结束日期
	Avatar      string `json:"avatar"`                         // 项目封面URL
}

// ProjectCreateReq 定义创建项目请求的结构体
type ProjectCreateReq struct {
	Name        string `json:"name" binding:"required"`        // 项目名称
	Description string `json:"description"`                    // 项目描述
	ActivityID  uint   `json:"activity_id" binding:"required"` // 关联的活动ID
	StartDate   int64  `json:"start_date" binding:"required"`  // 项目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`    // 项目结束日期
	Avatar      string `json:"avatar"`                         // 项目封面URL
}

// ProjectUpdateReq 定义更新项目请求的结构体，使用指针类型支持部分更新
type ProjectUpdateReq struct {
	Name        *string `json:"name"`        // 项目名称，可选
	Description *string `json:"description"` // 项目描述，可选
	ActivityID  *uint   `json:"activity_id"` // 关联的活动ID，可选
	StartDate   *int64  `json:"start_date"`  // 项目开始日期，可选
	EndDate     *int64  `json:"end_date"`    // 项目结束日期，可选
	Avatar      *string `json:"avatar"`      // 项目封面URL，可选
}

// ProjectResponse 定义项目响应结构体（不包含空的Activity字段）
type ProjectResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	ActivityID  uint   `json:"activity_id"`
	StartDate   int64  `json:"start_date"`
	EndDate     int64  `json:"end_date"`
	Avatar      string `json:"avatar"`
}

// CreateProject 处理创建项目请求
func CreateProject(c *gin.Context) {
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
	var req ProjectCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定创建项目请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 检查对应的活动是否存在且未被删除
	var activity model.Activity
	if err := database.DB.First(&activity, "id = ?", req.ActivityID).Error; err != nil {
		log.Warn("关联的活动不存在或已被删除", "activity_id", req.ActivityID)
		response.Fail(c, response.ErrNotFound.WithTips("关联的活动不存在或已被删除"))
		return
	}

	// 验证项目的时间范围是否在活动时间范围内
	if req.StartDate < activity.StartDate || req.EndDate > activity.EndDate {
		log.Warn("项目时间范围超出活动范围",
			"project_start", req.StartDate, "project_end", req.EndDate,
			"activity_start", activity.StartDate, "activity_end", activity.EndDate)
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目的开始和结束时间必须在活动时间范围内"))
		return
	}

	// 验证项目的开始时间不能晚于结束时间
	if req.StartDate >= req.EndDate {
		log.Warn("项目开始时间不能晚于或等于结束时间", "start_date", req.StartDate, "end_date", req.EndDate)
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目开始时间必须早于结束时间"))
		return
	}

	// 查询项目是否已存在
	var existingProject model.Project
	if err := database.DB.Where("owner_id = ? AND name = ?", StudentID, req.Name).First(&existingProject).Error; err == nil {
		log.Warn("项目已存在", "name", req.Name, "owner_id", StudentID)
		response.Fail(c, response.ErrAlreadyExists.WithTips("项目已存在"))
		return
	}

	// 创建项目模型实例
	project := model.Project{
		Name:        req.Name,
		Description: req.Description,
		ActivityID:  req.ActivityID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Avatar:      req.Avatar,
		OwnerID:     StudentID,
	}

	// 保存到数据库
	if err := database.DB.Create(&project).Error; err != nil {
		log.Error("创建项目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目创建成功", "project_id", project.ID, "owner_id", StudentID)
	response.Success(c, gin.H{
		"project_id": project.ID,
	})
}

// UpdateProject 处理更新项目请求
func UpdateProject(c *gin.Context) {
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
	var req ProjectUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定更新项目请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	var project model.Project
	if err := database.DB.Where("owner_id = ? AND id = ?", StudentID, c.Param("id")).First(&project).Error; err != nil {
		log.Error("查询项目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("项目未找到"))
		return
	}

	// 如果要更新ActivityID或时间字段，需要验证
	if req.ActivityID != nil || req.StartDate != nil || req.EndDate != nil {
		// 获取活动信息（如果更新了ActivityID，使用新的；否则使用原有的）
		activityID := project.ActivityID
		if req.ActivityID != nil {
			activityID = *req.ActivityID
		}

		var activity model.Activity
		if err := database.DB.First(&activity, "id = ?", activityID).Error; err != nil {
			log.Warn("关联的活动不存在或已被删除", "activity_id", activityID)
			response.Fail(c, response.ErrNotFound.WithTips("关联的活动不存在或已被删除"))
			return
		}

		// 计算更新后的时间范围
		startDate := project.StartDate
		endDate := project.EndDate
		if req.StartDate != nil {
			startDate = *req.StartDate
		}
		if req.EndDate != nil {
			endDate = *req.EndDate
		}

		// 验证项目的时间范围是否在活动时间范围内
		if startDate < activity.StartDate || endDate > activity.EndDate {
			log.Warn("项目时间范围超出活动范围",
				"project_start", startDate, "project_end", endDate,
				"activity_start", activity.StartDate, "activity_end", activity.EndDate)
			response.Fail(c, response.ErrInvalidRequest.WithTips("项目的开始和结束时间必须在活动时间范围内"))
			return
		}

		// 验证项目的开始时间不能晚于结束时间
		if startDate >= endDate {
			log.Warn("项目开始时间不能晚于或等于结束时间", "start_date", startDate, "end_date", endDate)
			response.Fail(c, response.ErrInvalidRequest.WithTips("项目开始时间必须早于结束时间"))
			return
		}
	}

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.ActivityID != nil {
		project.ActivityID = *req.ActivityID
	}
	if req.StartDate != nil {
		project.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		project.EndDate = *req.EndDate
	}
	if req.Avatar != nil {
		project.Avatar = *req.Avatar
	}

	if err := database.DB.Save(&project).Error; err != nil {
		log.Error("更新项目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目更新成功", "project_id", project.ID, "owner_id", StudentID)
	response.Success(c, project)
}

// DeleteProject 处理删除项目请求
func DeleteProject(c *gin.Context) {
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

	// 获取项目ID
	id := c.Param("id")
	if id == "" {
		log.Error("项目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目ID不能为空"))
		return
	}

	// 查询项目是否存在
	var project model.Project
	if err := database.DB.First(&project, "id = ? AND owner_id = ?", id, StudentID).Error; err != nil {
		log.Error("查询项目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("项目未找到"))
		return
	}

	if err := database.DB.Delete(&project).Error; err != nil {
		log.Error("删除项目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目删除成功", "project_id", project.ID, "owner_id", StudentID)
	response.Success(c)
}

// RestoreProject 处理恢复项目请求
func RestoreProject(c *gin.Context) {
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

	// 获取项目ID
	id := c.Param("id")
	if id == "" {
		log.Error("项目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目ID不能为空"))
		return
	}

	var project model.Project
	if err := database.DB.Unscoped().Where("id = ? AND owner_id = ?", id, StudentID).First(&project).Error; err != nil {
		log.Error("查询项目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("项目未找到"))
		return
	}

	project.DeletedAt = gorm.DeletedAt{}
	if err := database.DB.Unscoped().Save(&project).Error; err != nil {
		log.Error("恢复项目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目恢复成功", "project_id", project.ID, "owner_id", StudentID)
	response.Success(c)
}

// ListProjects 处理查询用户项目列表请求
func ListProjects(c *gin.Context) {
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

	var projects []model.Project
	// 查询项目时，同时确保关联的活动未被删除
	if err := database.DB.Joins("JOIN activity ON activity.id = project.activity_id AND activity.deleted_at IS NULL").
		Where("project.owner_id = ?", StudentID).
		Find(&projects).Error; err != nil {
		log.Error("查询项目列表失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 转换为响应格式，排除空的Activity字段
	var projectResponses []ProjectResponse
	for _, p := range projects {
		projectResponses = append(projectResponses, ProjectResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			OwnerID:     p.OwnerID,
			ActivityID:  p.ActivityID,
			StartDate:   p.StartDate,
			EndDate:     p.EndDate,
			Avatar:      p.Avatar,
		})
	}

	log.Info("查询项目列表成功", "count", len(projects), "owner_id", StudentID)
	response.Success(c, projectResponses)
}

// ColumnInProject 栏目信息结构体（用于项目详情返回）
type ColumnInProject struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type GetProjectResponse struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Avatar      string            `json:"avatar"`
	Description string            `json:"description"`
	StartDate   int64             `json:"start_date"`
	EndDate     int64             `json:"end_date"`
	Columns     []ColumnInProject `json:"columns"`
}

func GetProject(c *gin.Context) {

	// 获取项目ID
	id := c.Param("id")
	if id == "" {
		log.Error("项目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目ID不能为空"))
		return
	}

	var project model.Project
	if err := database.DB.Where("id = ?", id).First(&project).Error; err != nil {
		log.Error("查询项目失败", "error", err)
		response.Fail(c, response.ErrNotFound.WithTips("项目未找到"))
		return
	}

	// 查询该项目下的所有栏目
	var columns []model.Column
	if err := database.DB.Where("project_id = ?", project.ID).Find(&columns).Error; err != nil {
		log.Error("查询项目栏目失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 构建栏目响应数据
	var columnResponses []ColumnInProject
	for _, col := range columns {
		columnResponses = append(columnResponses, ColumnInProject{
			ID:     col.ID,
			Name:   col.Name,
			Avatar: col.Avatar,
		})
	}

	// 构建项目详情响应
	projectResponse := GetProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Avatar:      project.Avatar,
		Description: project.Description,
		StartDate:   project.StartDate,
		EndDate:     project.EndDate,
		Columns:     columnResponses,
	}

	response.Success(c, projectResponse)
}
