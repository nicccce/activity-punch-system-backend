package activity

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"activity-punch-system/tools"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Activity struct {
	Name        string `json:"name" binding:"required"`       // 项目名称
	OwnerID     string `json:"owner_id" binding:"required"`   // 项目创建人学号
	Description string `json:"description"`                   // 项目描述
	StartDate   int64  `json:"start_date" binding:"required"` // 项目开始日期
	EndDate     int64  `json:"end_date" binding:"required"`   // 项目结束日期
	Avatar      string `json:"avatar"`                        // 项目封面URL
}

// ActivityCreateReq 定义创建项目请求的结构体
type ActivityCreateReq struct {
	Name            string `json:"name" binding:"required,max=75"` // 项目名称
	Description     string `json:"description" binding:"max=200"`  // 项目描述
	StartDate       int64  `json:"start_date" binding:"required"`  // 项目开始日期
	EndDate         int64  `json:"end_date" binding:"required"`    // 项目结束日期
	Avatar          string `json:"avatar"`                         // 项目封面URL
	DailyPointLimit uint   `json:"daily_point_limit"`              // 每日积分上限，可选，0表示不限制
	CompletionBonus uint   `json:"completion_bonus"`               // 完成活动所有栏目后的额外奖励积分，可选，0表示无奖励
}

// ActivityUpdateReq 定义更新项目请求的结构体，使用指针类型支持部分更新
type ActivityUpdateReq struct {
	Name            *string `json:"name" binding:"omitempty,max=75"`         // 项目名称，可选
	Description     *string `json:"description" binding:"omitempty,max=200"` // 项目描述，可选
	StartDate       *int64  `json:"start_date"`                              // 项目开始日期，可选
	EndDate         *int64  `json:"end_date"`                                // 项目结束日期，可选
	Avatar          *string `json:"avatar"`                                  // 项目封面URL，可选
	DailyPointLimit *uint   `json:"daily_point_limit"`                       // 每日积分上限，可选，0表示不限制
	CompletionBonus *uint   `json:"completion_bonus"`                        // 完成活动所有栏目后的额外奖励积分，可选，0表示无奖励
}

// CreateActivity 处理创建项目请求
func CreateActivity(c *gin.Context) {
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
	var req ActivityCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定创建项目请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	var existingactivity model.Activity
	// 查询项目是否已存在
	err := database.DB.Where("name = ? AND start_date = ?", req.Name, req.StartDate).First(&existingactivity).Error
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

	activity := model.Activity{
		Name:            req.Name,
		OwnerID:         StudentID,
		Description:     req.Description,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Avatar:          req.Avatar,
		DailyPointLimit: req.DailyPointLimit,
		CompletionBonus: req.CompletionBonus,
	}

	if err := database.DB.Create(&activity).Error; err != nil {
		log.Error("创建项目失败", "error", err, "name", req.Name)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info(
		"项目创建成功",
		"name", req.Name,
		"owner_id", StudentID,
	)

	response.Success(c, gin.H{
		"activity_id": activity.ID,
	})
}

// ListActivitysReq 定义获取项目列表的查询参数结构体
type ListActivitysReq struct {
	OwnerID  string `form:"owner_id" json:"owner_id"`   // 项目创建人学号筛选
	Page     int    `form:"page" json:"page"`           // 页码，默认为1
	PageSize int    `form:"page_size" json:"page_size"` // 每页大小，默认为10
	Name     string `form:"name" json:"name"`           // 项目名称模糊查询
}

// ListActivitys 获取项目列表（支持查询参数）
func ListActivitys(c *gin.Context) {
	// 绑定查询参数到结构体
	var req ListActivitysReq
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
	query := database.DB.Model(&model.Activity{})

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

	var activitys []model.Activity
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("User").Offset(offset).Limit(req.PageSize).Find(&activitys).Error; err != nil {
		log.Error("获取项目列表失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 构建响应数据
	result := map[string]any{
		"activitys":   activitys,
		"total":       total,
		"page":        req.Page,
		"page_size":   req.PageSize,
		"total_pages": (total + int64(req.PageSize) - 1) / int64(req.PageSize),
	}

	log.Info("获取项目列表成功",
		"count", len(activitys),
		"total", total,
		"page", req.Page,
		"page_size", req.PageSize)

	response.Success(c, result)
}

// UpdateActivity 处理更新项目请求
func UpdateActivity(c *gin.Context) {
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
	var req ActivityUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定更新项目请求失败", "error", err, "id", id)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 查询项目是否存在
	var activity model.Activity
	if err := database.DB.First(&activity, "id = ?", id).Error; err != nil {
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
	if activity.OwnerID != StudentID {
		log.Warn("无权限更新项目", "id", id, "owner_id", activity.OwnerID, "student_id", StudentID)
		response.Fail(c, response.ErrForbidden.WithTips("无权限更新该项目"))
		return
	}

	if req.Name != nil {
		activity.Name = *req.Name
	}
	if req.Description != nil {
		activity.Description = *req.Description
	}
	if req.StartDate != nil {
		activity.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		activity.EndDate = *req.EndDate
	}
	if req.Avatar != nil {
		activity.Avatar = *req.Avatar
	}
	if req.DailyPointLimit != nil {
		activity.DailyPointLimit = *req.DailyPointLimit
	}
	if req.CompletionBonus != nil {
		activity.CompletionBonus = *req.CompletionBonus
	}

	if err := database.DB.Save(&activity).Error; err != nil {
		log.Error("更新项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("项目更新成功",
		"id", activity.ID,
		"name", activity.Name,
	)

	response.Success(c)
}

// DeleteActivity 处理删除活动请求（级联软删除关联的 Project 和 Column）
func DeleteActivity(c *gin.Context) {
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

	// 获取活动ID
	id := c.Param("id")
	if id == "" {
		log.Error("活动ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("活动ID不能为空"))
		return
	}

	// 查询活动是否存在
	var activity model.Activity
	if err := database.DB.First(&activity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("活动不存在", "id", id)
			response.Fail(c, response.ErrNotFound.WithTips("活动不存在"))
			return
		}
		log.Error("查询活动失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	if activity.OwnerID != StudentID {
		log.Warn("无权限删除活动", "id", id, "owner_id", activity.OwnerID, "student_id", StudentID)
		response.Fail(c, response.ErrForbidden.WithTips("无权限删除该活动"))
		return
	}

	// 使用事务进行级联删除
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 查询该活动下的所有项目ID
		var projectIDs []uint
		if err := tx.Model(&model.Project{}).
			Where("activity_id = ? AND deleted_at IS NULL", activity.ID).
			Pluck("id", &projectIDs).Error; err != nil {
			return err
		}

		// 2. 软删除这些项目下的所有栏目
		if len(projectIDs) > 0 {
			if err := tx.Where("project_id IN ? AND deleted_at IS NULL", projectIDs).
				Delete(&model.Column{}).Error; err != nil {
				return err
			}
			log.Info("级联删除栏目", "activity_id", activity.ID, "project_ids", projectIDs)
		}

		// 3. 软删除该活动下的所有项目
		if err := tx.Where("activity_id = ? AND deleted_at IS NULL", activity.ID).
			Delete(&model.Project{}).Error; err != nil {
			return err
		}
		log.Info("级联删除项目", "activity_id", activity.ID, "project_count", len(projectIDs))

		// 4. 软删除活动本身
		if err := tx.Delete(&activity).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("删除活动失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("活动删除成功（含级联删除）",
		"id", activity.ID,
	)

	response.Success(c)
}

// RestoreActivity 处理还原删除的活动请求（级联恢复关联的 Project 和 Column）
func RestoreActivity(c *gin.Context) {
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

	// 获取活动ID
	id := c.Param("id")
	if id == "" {
		log.Error("活动ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("活动ID不能为空"))
		return
	}

	// 查询活动是否存在（包括已删除的）
	var activity model.Activity
	if err := database.DB.Unscoped().First(&activity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("活动不存在", "id", id)
			response.Fail(c, response.ErrNotFound.WithTips("活动不存在"))
			return
		}
		log.Error("查询活动失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	if activity.OwnerID != StudentID {
		log.Warn("无权限还原活动", "id", id, "owner_id", activity.OwnerID, "student_id", StudentID)
		response.Fail(c, response.ErrForbidden.WithTips("无权限还原该活动"))
		return
	}

	// 使用事务进行级联恢复
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 恢复活动本身
		if err := tx.Unscoped().Model(&activity).Update("deleted_at", nil).Error; err != nil {
			return err
		}

		// 2. 查询该活动下所有已删除的项目ID
		var projectIDs []uint
		if err := tx.Unscoped().Model(&model.Project{}).
			Where("activity_id = ? AND deleted_at IS NOT NULL", activity.ID).
			Pluck("id", &projectIDs).Error; err != nil {
			return err
		}

		// 3. 恢复这些项目下所有已删除的栏目
		if len(projectIDs) > 0 {
			if err := tx.Unscoped().Model(&model.Column{}).
				Where("project_id IN ? AND deleted_at IS NOT NULL", projectIDs).
				Update("deleted_at", nil).Error; err != nil {
				return err
			}
			log.Info("级联恢复栏目", "activity_id", activity.ID, "project_ids", projectIDs)
		}

		// 4. 恢复该活动下所有已删除的项目
		if err := tx.Unscoped().Model(&model.Project{}).
			Where("activity_id = ? AND deleted_at IS NOT NULL", activity.ID).
			Update("deleted_at", nil).Error; err != nil {
			return err
		}
		log.Info("级联恢复项目", "activity_id", activity.ID, "project_count", len(projectIDs))

		return nil
	})

	if err != nil {
		log.Error("还原活动失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	log.Info("活动还原成功（含级联恢复）",
		"id", activity.ID,
		"name", activity.Name,
	)
	response.Success(c)
}

type ProjectInActivity struct {
	ID          uint   `json:"id"`          // 项目ID
	Name        string `json:"name"`        // 项目名称
	Avatar      string `json:"avatar"`      // 项目封面URL
	Description string `json:"description"` // 项目描述
}

// GetActivity 获取单个项目详情
func GetActivity(c *gin.Context) {
	// 获取项目ID
	id := c.Param("id")
	if id == "" {
		log.Error("项目ID不能为空")
		response.Fail(c, response.ErrInvalidRequest.WithTips("项目ID不能为空"))
		return
	}

	var activity model.Activity
	// 查询项目详情
	if err := database.DB.Preload("User").First(&activity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("项目不存在", "id", id)
			response.Fail(c, response.ErrNotFound.WithTips("项目不存在"))
			return
		}
		log.Error("查询项目失败", "error", err, "id", id)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 查询关联的所有项目信息
	var projectsInActivity []ProjectInActivity
	if err := database.DB.Model(&model.Project{}).
		Select("ID, name, avatar, description").
		Where("activity_id = ?", activity.ID).
		Find(&projectsInActivity).Error; err != nil {
		log.Error("查询项目关联信息失败", "error", err, "activity_id", activity.ID)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 构建响应数据
	result := gin.H{
		"activity": activity,
		"projects": projectsInActivity,
	}

	log.Info("获取项目详情成功",
		"id", activity.ID,
		"name", activity.Name,
		"projects_count", len(projectsInActivity),
	)

	response.Success(c, result)
}
func MineActivities(c *gin.Context) {
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
	offset, limit := tools.GetPage(c)
	var activities []struct {
		model.Model
		Name        string `gorm:"type:varchar(100);not null" json:"name"` // 活动名称
		Description string `gorm:"type:varchar(255);" json:"description"`  // 活动描述
		StartDate   int64  `gorm:"" json:"start_date"`                     // 活动开始时间
		EndDate     int64  `gorm:"" json:"end_date"`                       // 活动结束时间
		Avatar      string `gorm:"type:varchar(255);" json:"avatar"`       // 活动封面URL
	}
	if err := database.DB.Table("activity").Where("owner_id = ?", StudentID).
		Offset(offset).Limit(limit).
		Find(&activities).Error; err != nil {
		log.Error("查询管理员自己创建的所有活动失败", "error", err, "student_id", StudentID)
		response.Fail(c, response.ErrDatabase)
		return
	}
	response.Success(c, activities)
}
