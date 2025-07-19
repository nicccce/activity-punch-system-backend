package project

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Project struct {
	Name        string `json:"name" binding:"required"`       // 项目名称
	OwnerID     string `json:"owner_id" binding:"required"`   // 项目创建人学号
	Description string `json:"description"`                   // 项目描述
	Catagory    string `json:"category" binding:"required"`   // 项目归属（暑期打卡活动、寒假打卡活动等）
	StartDate   int64  `json:"start_date" binding:"required"` // 项目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`   // 项目结束日期
}

// ProjectCreateReq 定义创建项目请求的结构体
type ProjectCreateReq struct {
	Name        string `json:"name" binding:"required"`       // 项目名称
	Description string `json:"description"`                   // 项目描述
	Category    string `json:"category" binding:"required"`   // 项目归属（暑期打卡活动、寒假打卡活动等）
	StartDate   int64  `json:"start_date" binding:"required"` // 项目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`   // 项目结束日期
}

// ProjectUpdateReq 定义更新项目请求的结构体，使用指针类型支持部分更新
type ProjectUpdateReq struct {
	Name        *string `json:"name"`        // 项目名称，可选
	Description *string `json:"description"` // 项目描述，可选
	Category    *string `json:"category"`    // 项目归属，可选
	StartDate   *int64  `json:"start_date"`  // 项目开始日期，可选
	EndDate     *int64  `json:"end_date"`    // 项目结束日期，可选
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

	var existingproject model.Project
	// 查询项目是否已存在
	err := database.DB.Where("name = ? AND start_date = ?", req.Name, req.StartDate).First(&existingproject).Error
	if err == nil {
		// 项目已存在
		log.Warn("项目已存在", "name", req.Name, "start_date", req.StartDate)
		response.Fail(c, response.ErrAlreadyExists.WithTips("项目已存在"))
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 数据库错误
		log.Error("数据库查询失败", "error", err, "name", req.Name, "start_date", req.StartDate)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	project := model.Project{
		Name:        req.Name,
		OwnerID:     StudentID,
		Description: req.Description,
		Category:    req.Category,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		log.Error("创建项目失败", "error", err, "name", req.Name)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info(
		"项目创建成功",
		"name", req.Name,
		"owner_id", StudentID,
	)

	response.Success(c)
}

// ListProjectsReq 定义获取项目列表的查询参数结构体
type ListProjectsReq struct {
	Category string `form:"category" json:"category"`   // 项目分类筛选
	OwnerID  string `form:"owner_id" json:"owner_id"`   // 项目创建人学号筛选
	Page     int    `form:"page" json:"page"`           // 页码，默认为1
	PageSize int    `form:"page_size" json:"page_size"` // 每页大小，默认为10
	Name     string `form:"name" json:"name"`           // 项目名称模糊查询
}

// ListProjects 获取项目列表（支持查询参数）
func ListProjects(c *gin.Context) {
	// 绑定查询参数到结构体
	var req ListProjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Error("绑定查询参数失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 构建查询条件
	query := database.DB.Model(&model.Project{})

	// 根据分类筛选
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	// 根据创建人学号筛选
	if req.OwnerID != "" {
		query = query.Where("owner_id = ?", req.OwnerID)
	}

	// 根据项目名称模糊查询
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("获取项目总数失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 分页查询
	var projects []model.Project
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&projects).Error; err != nil {
		log.Error("获取项目列表失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 构建响应数据
	result := map[string]interface{}{
		"projects":    projects,
		"total":       total,
		"page":        req.Page,
		"page_size":   req.PageSize,
		"total_pages": (total + int64(req.PageSize) - 1) / int64(req.PageSize),
	}

	log.Info("获取项目列表成功",
		"count", len(projects),
		"total", total,
		"page", req.Page,
		"page_size", req.PageSize)

	response.Success(c, result)
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
	// 获取项目ID
	id := c.Param("id")
	if id == "" {
		log.Error("项目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目ID不能为空"))
		return
	}

	// 定义请求结构体并绑定 JSON 数据
	var req ProjectUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定更新项目请求失败", "error", err, "id", id)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 查询项目是否存在
	var project model.Project
	if err := database.DB.First(&project, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("项目不存在", "id", id)
			response.Fail(c, response.ErrNotFound.WithTips("项目不存在"))
			return
		}
		log.Error("查询项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 权限检查
	if project.OwnerID != StudentID {
		log.Warn("无权限更新项目", "id", id, "owner_id", project.OwnerID, "student_id", StudentID)
		response.Fail(c, response.ErrForbidden.WithTips("无权限更新该项目"))
		return
	}

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Category != nil {
		project.Category = *req.Category
	}
	if req.StartDate != nil {
		project.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		project.EndDate = *req.EndDate
	}

	if err := database.DB.Save(&project).Error; err != nil {
		log.Error("更新项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目更新成功",
		"id", project.ID,
		"name", project.Name,
	)

	response.Success(c)
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
	if err := database.DB.First(&project, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("项目不存在", "id", id)
			response.Fail(c, response.ErrNotFound.WithTips("项目不存在"))
			return
		}
		log.Error("查询项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	if project.OwnerID != StudentID {
		log.Warn("无权限删除项目", "id", id, "owner_id", project.OwnerID, "student_id", StudentID)
		response.Fail(c, response.ErrForbidden.WithTips("无权限删除该项目"))
		return
	}

	// 删除项目
	if err := database.DB.Delete(&project).Error; err != nil {
		log.Error("删除项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目删除成功",
		"id", project.ID,
	)

	response.Success(c)
}

// RestoreProject 处理还原删除的项目请求
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

	// 查询项目是否存在
	var project model.Project
	if err := database.DB.Unscoped().First(&project, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("项目不存在", "id", id)
			response.Fail(c, response.ErrNotFound.WithTips("项目不存在"))
			return
		}
		log.Error("查询项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	if project.OwnerID != StudentID {
		log.Warn("无权限还原项目", "id", id, "owner_id", project.OwnerID, "student_id", StudentID)
		response.Fail(c, response.ErrForbidden.WithTips("无权限还原该项目"))
		return
	}

	// 还原项目（使用 Unscoped 和 Model 的 DeletedAt 字段来恢复）
	project.DeletedAt = gorm.DeletedAt{}
	if err := database.DB.Unscoped().Save(&project).Error; err != nil {
		log.Error("还原项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目还原成功",
		"id", project.ID,
		"name", project.Name,
	)
	response.Success(c)
}
