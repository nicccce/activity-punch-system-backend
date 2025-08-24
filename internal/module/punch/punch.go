package punch

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PunchInsertRequest 定义插入打卡记录的请求体结构
type PunchInsertRequest struct {
	ColumnID int      `json:"column_id" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Images   []string `json:"images" binding:"omitempty"`
}

type PunchWithImgs struct {
	model.Punch
	Imgs []string `json:"imgs"`
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

	// 绑定 JSON 数据
	var req PunchInsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
	startDate, _ := time.Parse("20060102", startDateStr)
	endDate, _ := time.Parse("20060102", endDateStr)
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

	// 处理图片URL保存到punch_img表
	if len(req.Images) > 0 {
		for _, imgUrl := range req.Images {
			punchImg := &model.PunchImg{
				PunchID:  punch.ID,
				ColumnID: req.ColumnID,
				ImgURL:   imgUrl,
			}
			if err := database.DB.Create(punchImg).Error; err != nil {
				log.Error("插入打卡图片记录失败", "error", err)
				continue
			}
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

	// 查询图片数组
	var imgs []model.PunchImg
	database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
	imgUrls := make([]string, 0, len(imgs))
	for _, img := range imgs {
		imgUrls = append(imgUrls, img.ImgURL)
	}

	response.Success(c, PunchWithImgs{
		Punch: punch,
		Imgs:  imgUrls,
	})
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

	// 今日是否已打卡
	today := time.Now().Truncate(24 * time.Hour) // 今日零点时间
	hasPunchedToday := false
	for _, punch := range punches {
		if punch.CreatedAt.After(today) || punch.CreatedAt.Equal(today) {
			hasPunchedToday = true
			break
		}
	}

	// 查询每条打卡记录的图片

	var result []PunchWithImgs
	for _, punch := range punches {
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}
		result = append(result, PunchWithImgs{
			Punch: punch,
			Imgs:  imgUrls,
		})
	}

	// 查询该栏目下不同 user_id 数量
	var userCount int64
	database.DB.Model(&model.Punch{}).Where("column_id = ? ", columnIDStr).Distinct("user_id").Count(&userCount)

	// 查询当前用户打卡数量
	var myCount int64
	database.DB.Model(&model.Punch{}).Where("column_id = ? AND user_id = ? ", columnIDStr, studentID).Count(&myCount)

	response.Success(c, gin.H{
		"records":       result,
		"user_count":    userCount,
		"my_count":      myCount,
		"punched_today": hasPunchedToday,
	})
}

// DeletePunch 删除自己拥有的打卡记录
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

	// 判断打卡时间是否在栏目时间范围内
	startDateStr := strconv.FormatInt(column.StartDate, 10)
	endDateStr := strconv.FormatInt(column.EndDate, 10)
	startDate, _ := time.Parse("20060102", startDateStr)
	endDate, _ := time.Parse("20060102", endDateStr)
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

// PunchUpdateRequest 修改打卡请求体
type PunchUpdateRequest struct {
	ColumnID int      `json:"column_id" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Images   []string `json:"images" binding:"omitempty"`
}

// UpdatePunch 修改打卡记录
func UpdatePunch(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
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

	var req PunchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	var punch model.Punch
	if err := database.DB.First(&punch, "id = ? AND user_id = ?", idStr, studentID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("打卡记录不存在或无权限"))
		return
	}

	var column model.Column
	if err := database.DB.First(&column, "id = ?", req.ColumnID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在"))
		return
	}

	startDateStr := strconv.FormatInt(column.StartDate, 10)
	endDateStr := strconv.FormatInt(column.EndDate, 10)
	startDate, _ := time.Parse("20060102", startDateStr)
	endDate, _ := time.Parse("20060102", endDateStr)
	if punch.CreatedAt.Before(startDate) || punch.CreatedAt.After(endDate) {
		response.Fail(c, response.ErrInvalidRequest.WithTips("打卡时间不在栏目时间范围内，无法修改"))
		return
	}

	// 修改打卡内容
	punch.Content = req.Content
	punch.ColumnID = req.ColumnID
	if err := database.DB.Save(&punch).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 可选：处理图片（如需覆盖原图片，可先删除原图片再插入新图片）
	if len(req.Images) > 0 {
		// 删除原图片
		database.DB.Where("punch_id = ?", punch.ID).Delete(&model.PunchImg{})
		for _, imgUrl := range req.Images {
			punchImg := &model.PunchImg{
				PunchID:  punch.ID,
				ColumnID: req.ColumnID,
				ImgURL:   imgUrl,
			}
			database.DB.Create(punchImg)
		}
	}

	// 查询图片数组
	var imgs []model.PunchImg
	database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
	imgUrls := make([]string, 0, len(imgs))
	for _, img := range imgs {
		imgUrls = append(imgUrls, img.ImgURL)
	}

	response.Success(c, struct {
		model.Punch
		Imgs []string `json:"imgs"`
	}{
		Punch: punch,
		Imgs:  imgUrls,
	})
}

// 获取待审核打卡列表
func GetPendingPunchList(c *gin.Context) {
	var punches []model.Punch
	if err := database.DB.Where("status = 0").Order("created_at desc").Find(&punches).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	var result []PunchWithImgs
	for _, punch := range punches {
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}
		result = append(result, PunchWithImgs{
			Punch: punch,
			Imgs:  imgUrls,
		})
	}
	response.Success(c, result)
}

// 查询自己所有打卡记录
func GetMyPunchList(c *gin.Context) {
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
	if err := database.DB.Where("user_id = ?", studentID).Order("created_at desc").Find(&punches).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	var result []PunchWithImgs
	for _, punch := range punches {
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}
		result = append(result, PunchWithImgs{
			Punch: punch,
			Imgs:  imgUrls,
		})
	}
	response.Success(c, result)
}

// 获取最近参与栏目、项目、活动
func GetRecentParticipation(c *gin.Context) {
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
	database.DB.Where("user_id = ?", studentID).Order("created_at desc").Find(&punches)

	columnMap := make(map[int]bool)
	projectMap := make(map[int]bool)
	activityMap := make(map[uint]bool)
	var recentColumns []model.Column
	var recentProjects []model.Project
	var recentActivities []model.Activity
	var punchResults []PunchWithImgs

	for _, punch := range punches {
		// 查图片
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}
		punchResults = append(punchResults, PunchWithImgs{
			Punch: punch,
			Imgs:  imgUrls,
		})
		// 1. 查找栏目
		if !columnMap[punch.ColumnID] {
			var col model.Column
			if err := database.DB.First(&col, "id = ?", punch.ColumnID).Error; err == nil {
				recentColumns = append(recentColumns, col)
				columnMap[punch.ColumnID] = true
				// 2. 查找项目
				if !projectMap[int(col.ProjectID)] && col.ProjectID != 0 {
					var proj model.Project
					if err := database.DB.First(&proj, "id = ?", col.ProjectID).Error; err == nil {
						recentProjects = append(recentProjects, proj)
						projectMap[int(col.ProjectID)] = true
						// 3. 查找活动
						if !activityMap[proj.ActivityID] && proj.ActivityID != 0 {
							var act model.Activity
							if err := database.DB.First(&act, "id = ?", proj.ActivityID).Error; err == nil {
								recentActivities = append(recentActivities, act)
								activityMap[proj.ActivityID] = true
							}
						}
					}
				}
			}
		}
	}

	response.Success(c, gin.H{
		"punches":    punchResults,
		"columns":    recentColumns,
		"projects":   recentProjects,
		"activities": recentActivities,
	})
}
