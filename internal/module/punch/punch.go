package punch

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/pictureBed"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"errors"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// 北京时区
var beijingLocation = time.FixedZone("CST", 8*60*60)

// getTodayStart 获取北京时间今日零点
func getTodayStart() time.Time {
	now := time.Now().In(beijingLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, beijingLocation)
}

// PunchInsertRequest 定义插入打卡记录的请求体结构
type PunchInsertRequest struct {
	ColumnID int      `json:"column_id" binding:"required"`
	Content  string   `json:"content" binding:"required,max=1000"`
	Images   []string `json:"images" binding:"omitempty,max=9"`
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

	// 绑定 JSON 数据
	var req PunchInsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定打卡请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 验证栏目ID不能为空或小于等于0
	if req.ColumnID <= 0 {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目ID不能为空"))
		return
	}
	today := getTodayStart()
	count := int64(0)
	// 统计今日打卡次数：包含未删除的所有记录 + 已删除但审核不通过的记录（防止删除后重新打卡绕过限制）
	if err := database.DB.Table("punch").
		Where("user_id = ? AND column_id = ? AND created_at >= ?", userPayload.ID, req.ColumnID, today).
		Where("deleted_at IS NULL OR status = 2").
		Count(&count).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 查询栏目每日打卡限制
	var columnLimit int64
	if err := database.DB.Model(&model.Column{}).Select("daily_punch_limit").Where("id = ?", req.ColumnID).Scan(&columnLimit).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// columnLimit > 0 表示有设置每日打卡次数限制，0 表示不限制
	if columnLimit > 0 && count >= columnLimit {
		response.Fail(c, response.ErrInvalidRequest.WithTips("今日已达到打卡次数上限，无法继续打卡"))
		return
	}

	// 获取栏目时间范围，判断是否允许打卡
	var column model.Column
	if err := database.DB.Preload("Project").Preload("Project.Activity").First(&column, "id = ?", req.ColumnID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在"))
		return
	}
	startDateStr := strconv.FormatInt(column.StartDate, 10)
	endDateStr := strconv.FormatInt(column.EndDate, 10)
	startDate, _ := time.Parse("20060102", startDateStr)
	endDate, _ := time.Parse("20060102", endDateStr)
	currentTime := time.Now()

	// 判断当前日期是否在栏目时间范围内
	if currentTime.Before(startDate) || currentTime.After(endDate) {
		response.Fail(c, response.ErrInvalidRequest.WithTips("当前时间不在栏目时间范围内，无法打卡"))
		return
	}

	// 判断当前时间是否在每日打卡时间范围内
	if column.StartTime != "" && column.EndTime != "" {
		now := time.Now()
		currentTimeStr := now.Format("15:04") // HH:MM 格式

		// 解析每日开始和结束时间
		startTime, err1 := time.Parse("15:04", column.StartTime)
		endTime, err2 := time.Parse("15:04", column.EndTime)
		currentParsed, err3 := time.Parse("15:04", currentTimeStr)

		if err1 != nil || err2 != nil || err3 != nil {
			response.Fail(c, response.ErrInvalidRequest.WithTips("时间格式错误"))
			return
		}

		// 处理跨天情况（例如 22:00 - 06:00）
		if endTime.Before(startTime) {
			// 跨天情况：当前时间在开始时间之后或结束时间之前
			if currentParsed.Before(startTime) && currentParsed.After(endTime) {
				response.Fail(c, response.ErrInvalidRequest.WithTips("当前时间不在打卡时间范围内，无法打卡"))
				return
			}
		} else {
			// 不跨天情况：当前时间必须在开始和结束时间之间
			if currentParsed.Before(startTime) || currentParsed.After(endTime) {
				response.Fail(c, response.ErrInvalidRequest.WithTips("当前时间不在打卡时间范围内，无法打卡"))
				return
			}
		}
	}

	punch := &model.Punch{
		ColumnID: req.ColumnID,
		UserID:   userPayload.ID,
		Content:  req.Content,
		Status:   0, // 默认待审核
	}
	tx := database.DB.WithContext(context.WithValue(context.Background(), "fk_user_activity", &model.FkUserActivity{
		ActivityID: column.Project.Activity.ID,
		UserID:     userPayload.ID,
	}))
	if err := tx.Create(punch).Error; err != nil {
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
	PunchID    int    `json:"punch_id" binding:"required"`
	Status     int    `json:"status" binding:"required"` // 1: 通过, 2: 拒绝
	Special    bool   `json:"special"`                   // 是否特殊打分
	Score      int    `json:"score"`
	Cause      string `json:"cause"`
	MarkedBy   string `json:"marked_by"`   // 审核人
	ClearScore bool   `json:"clear_score"` // 是否清除之前这条punch的分数(如果0或2的话
}
type reviewRes struct {
	PunchID          int  `json:"punch_id"`
	Status           int  `json:"status"`
	AddedScore       int  `json:"added_score"`
	ProjectBonus     int  `json:"project_bonus"`     // 项目完成奖励积分
	ActivityBonus    int  `json:"activity_bonus"`    // 活动完成奖励积分
	DailyLimitHit    bool `json:"daily_limit_hit"`   // 是否触发每日积分上限
	ProjectComplete  bool `json:"project_complete"`  // 是否完成项目所有栏目
	ActivityComplete bool `json:"activity_complete"` // 是否完成活动所有栏目
}

// getTodayPointsForActivity 获取用户今日在活动中已获得的积分（排除不计入上限的项目和特殊栏目）
func getTodayPointsForActivity(userID uint, activityID uint) (uint, error) {
	today := getTodayStart()
	var totalPoints uint

	// 查询今日获得的积分，排除 exempt_from_limit = true 的项目和 optional = true 的特殊栏目
	err := database.DB.Table("score").
		Select("COALESCE(SUM(score.count), 0)").
		Joins("JOIN `column` ON score.column_id = `column`.id").
		Joins("JOIN project ON `column`.project_id = project.id").
		Where("score.user_id = ? AND project.activity_id = ? AND score.created_at >= ? AND score.deleted_at IS NULL AND project.exempt_from_limit = ? AND `column`.optional = ?",
			userID, activityID, today, false, false).
		Scan(&totalPoints).Error

	return totalPoints, err
}

// checkProjectCompletion 检查用户今日是否完成了项目下所有必需栏目的打卡（排除特殊栏目）
func checkProjectCompletion(userID uint, projectID uint) (bool, error) {
	today := getTodayStart()

	// 获取项目下所有必需栏目数量（排除 optional = true 的特殊栏目）
	var totalColumns int64
	if err := database.DB.Model(&model.Column{}).Where("project_id = ? AND deleted_at IS NULL AND optional = ?", projectID, false).Count(&totalColumns).Error; err != nil {
		return false, err
	}

	if totalColumns == 0 {
		return false, nil
	}

	// 获取用户今日已打卡且审核通过的必需栏目数量（去重，排除特殊栏目）
	var punchedColumns int64
	if err := database.DB.Table("punch").
		Select("COUNT(DISTINCT column_id)").
		Joins("JOIN `column` ON punch.column_id = `column`.id").
		Where("punch.user_id = ? AND `column`.project_id = ? AND punch.created_at >= ? AND punch.status = 1 AND punch.deleted_at IS NULL AND `column`.optional = ?",
			userID, projectID, today, false).
		Scan(&punchedColumns).Error; err != nil {
		return false, err
	}

	return punchedColumns >= totalColumns, nil
}

// checkActivityCompletion 检查用户今日是否完成了活动下所有必需栏目的打卡（排除特殊栏目）
func checkActivityCompletion(userID uint, activityID uint) (bool, error) {
	today := getTodayStart()

	// 获取活动下所有必需栏目数量（通过项目关联，排除 optional = true 的特殊栏目）
	var totalColumns int64
	if err := database.DB.Table("`column`").
		Joins("JOIN project ON `column`.project_id = project.id").
		Where("project.activity_id = ? AND `column`.deleted_at IS NULL AND project.deleted_at IS NULL AND `column`.optional = ?", activityID, false).
		Count(&totalColumns).Error; err != nil {
		return false, err
	}

	if totalColumns == 0 {
		return false, nil
	}

	// 获取用户今日已打卡且审核通过的必需栏目数量（去重，排除特殊栏目）
	var punchedColumns int64
	if err := database.DB.Table("punch").
		Select("COUNT(DISTINCT punch.column_id)").
		Joins("JOIN `column` ON punch.column_id = `column`.id").
		Joins("JOIN project ON `column`.project_id = project.id").
		Where("punch.user_id = ? AND project.activity_id = ? AND punch.created_at >= ? AND punch.status = 1 AND punch.deleted_at IS NULL AND `column`.optional = ?",
			userID, activityID, today, false).
		Scan(&punchedColumns).Error; err != nil {
		return false, err
	}

	return punchedColumns >= totalColumns, nil
}

// hasReceivedProjectCompletionBonus 检查用户今日是否已领取过项目完成奖励
func hasReceivedProjectCompletionBonus(userID uint, projectID uint) (bool, error) {
	today := getTodayStart()
	var count int64

	err := database.DB.Model(&model.Score{}).
		Where("user_id = ? AND cause = ? AND created_at >= ? AND deleted_at IS NULL",
			userID, fmt.Sprintf("ProjectCompletionBonus#%d", projectID), today).
		Count(&count).Error

	return count > 0, err
}

// hasReceivedActivityCompletionBonus 检查用户今日是否已领取过活动完成奖励
func hasReceivedActivityCompletionBonus(userID uint, activityID uint) (bool, error) {
	today := getTodayStart()
	var count int64

	err := database.DB.Model(&model.Score{}).
		Where("user_id = ? AND cause = ? AND created_at >= ? AND deleted_at IS NULL",
			userID, fmt.Sprintf("ActivityCompletionBonus#%d", activityID), today).
		Count(&count).Error

	return count > 0, err
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

	// 验证status值是否有效
	if req.Status < 0 || req.Status > 2 {
		response.Fail(c, response.ErrInvalidRequest.WithTips("状态值无效，只能为0(待审核)、1(通过)、2(拒绝)"))
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
	res := reviewRes{
		PunchID:    req.PunchID,
		Status:     req.Status,
		AddedScore: 0,
	}
	//获取方式可优化(优化为前端传来
	var projectID uint
	database.DB.Table("column").Select("project_id").Where("id = ?", punch.ColumnID).Scan(&projectID)

	// 获取完整的项目信息（包含CompletionBonus和ExemptFromLimit）
	var project model.Project
	if err := database.DB.First(&project, projectID).Error; err != nil {
		c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但打分失败 未找到所属的project", Data: res})
		return
	}
	activityID := project.ActivityID

	// 获取完整的活动信息（包含DailyPointLimit）
	var activity model.Activity
	if err := database.DB.First(&activity, activityID).Error; err != nil {
		c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但打分失败 未找到所属的activity", Data: res})
		return
	}

	tx := database.DB.WithContext(context.WithValue(context.Background(), "fk_user_activity", &model.FkUserActivity{
		ActivityID: activityID,
		UserID:     punch.UserID, // 使用打卡者的ID，而非审核者的ID
	}))

	// 辅助函数：检查每日积分上限并发放积分
	awardScore := func(scoreToAward int, cause string) (bool, string) {
		// 如果活动设置了每日积分上限，且该项目不豁免
		if activity.DailyPointLimit > 0 && !project.ExemptFromLimit {
			currentPoints, err := getTodayPointsForActivity(punch.UserID, activityID)
			if err != nil {
				return false, "检查每日积分上限失败"
			}
			if currentPoints >= activity.DailyPointLimit {
				res.DailyLimitHit = true
				return false, fmt.Sprintf("已达到每日积分上限(%d分)", activity.DailyPointLimit)
			}
			// 如果加上这次分数会超过上限，只给到上限
			if currentPoints+uint(scoreToAward) > activity.DailyPointLimit {
				scoreToAward = int(activity.DailyPointLimit - currentPoints)
				res.DailyLimitHit = true
			}
		}

		score := model.Score{
			UserID:   punch.UserID,
			Count:    uint(scoreToAward),
			Cause:    cause,
			MarkedBy: fmt.Sprintf("%s#%d", req.MarkedBy, userPayload.ID),
			PunchID:  punch.ID,
			ColumnID: uint(punch.ColumnID),
		}
		if err := tx.Create(&score).Error; err != nil {
			return false, "插入打分记录失败"
		}
		res.AddedScore = scoreToAward
		return true, ""
	}

	// 辅助函数：检查并发放项目完成奖励
	checkAndAwardProjectBonus := func() {
		if project.CompletionBonus == 0 {
			return
		}
		// 检查是否完成了项目所有栏目
		complete, err := checkProjectCompletion(punch.UserID, projectID)
		if err != nil || !complete {
			return
		}
		res.ProjectComplete = true

		// 检查今日是否已经领取过奖励
		received, err := hasReceivedProjectCompletionBonus(punch.UserID, projectID)
		if err != nil || received {
			return
		}

		// 发放项目完成奖励
		bonusScore := model.Score{
			UserID:   punch.UserID,
			Count:    project.CompletionBonus,
			Cause:    fmt.Sprintf("ProjectCompletionBonus#%d", projectID),
			MarkedBy: fmt.Sprintf("%s#%d", req.MarkedBy, userPayload.ID),
			PunchID:  punch.ID,
			ColumnID: uint(punch.ColumnID),
		}
		if err := tx.Create(&bonusScore).Error; err != nil {
			log.Warn("发放项目完成奖励失败", "err", err.Error())
			return
		}
		res.ProjectBonus = int(project.CompletionBonus)
	}

	// 辅助函数：检查并发放活动完成奖励
	checkAndAwardActivityBonus := func() {
		if activity.CompletionBonus == 0 {
			return
		}
		// 检查是否完成了活动所有栏目
		complete, err := checkActivityCompletion(punch.UserID, activityID)
		if err != nil || !complete {
			return
		}
		res.ActivityComplete = true

		// 检查今日是否已经领取过奖励
		received, err := hasReceivedActivityCompletionBonus(punch.UserID, activityID)
		if err != nil || received {
			return
		}

		// 发放活动完成奖励
		bonusScore := model.Score{
			UserID:   punch.UserID,
			Count:    activity.CompletionBonus,
			Cause:    fmt.Sprintf("ActivityCompletionBonus#%d", activityID),
			MarkedBy: fmt.Sprintf("%s#%d", req.MarkedBy, userPayload.ID),
			PunchID:  punch.ID,
			ColumnID: uint(punch.ColumnID),
		}
		if err := tx.Create(&bonusScore).Error; err != nil {
			log.Warn("发放活动完成奖励失败", "err", err.Error())
			return
		}
		res.ActivityBonus = int(activity.CompletionBonus)
	}

	// 大粪！通过才考虑打分
	if req.Status == 1 {

		if req.Special {
			if req.Score <= 0 {
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但自定义打分失败 分数不能小于1", Data: res})
				return
			}
			if req.Cause == "Auto" {
				req.Cause += "#并非自动打分"
			}

			ok, errMsg := awardScore(req.Score, req.Cause)
			if !ok {
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但自定义打分失败: " + errMsg, Data: res})
				return
			}

			// 检查并发放项目完成奖励
			checkAndAwardProjectBonus()
			// 检查并发放活动完成奖励
			checkAndAwardActivityBonus()

		} else {
			//自动打分
			//不可重复
			exist := false
			if err := database.DB.
				Raw("SELECT EXISTS(SELECT 1 FROM score WHERE user_id = ? AND punch_id = ? AND deleted_at IS NULL)",
					userPayload.ID, punch.ID).
				Scan(&exist).Error; err != nil {
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但自动打分查重时失败", Data: res})
				return
			}
			if exist {
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 自动打分失败,因为此前已经打过分,若需要加分,尝试special=true", Data: res})
				return
			}
			if err := tx.Table("column").Select("point_earned").Where("id = ?", punch.ColumnID).Scan(&(req.Score)).Error; err != nil {
				log.Warn("数据库 自动打分时获取column设置的分数时失败", "err", err.Error())
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但自动打分失败", Data: res})
				return
			}

			ok, errMsg := awardScore(req.Score, "Auto")
			if !ok {
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但自动打分失败: " + errMsg, Data: res})
				return
			}

			// 检查并发放项目完成奖励
			checkAndAwardProjectBonus()
			// 检查并发放活动完成奖励
			checkAndAwardActivityBonus()
		}
	} else if req.ClearScore && req.Status == 2 { //扣粪!
		var scores []model.Score
		if err := tx.Where("user_id = ? AND punch_id = ?", userPayload.ID, punch.ID).Find(&scores).Error; err != nil {
			log.Warn("数据库 扣分时获取score记录失败", "err", err.Error())
			c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但扣分失败", Data: res})
			return
		}
		for _, s := range scores {
			if err := tx.Delete(&s).Error; err != nil {
				log.Warn("数据库 扣分时删除score记录发生错误!", "err", err.Error())
				c.JSON(206, response.ResponseBody{Code: 206, Msg: "已审核 但扣分未完全完成", Data: res})
				return
			}
			res.AddedScore -= int(s.Count)
		}

	}
	response.Success(c, res)
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

	var punches []model.Punch
	// 查询当前用户未被删除的打卡记录（使用 userPayload.ID 而非 StudentID）
	if err := database.DB.Where("column_id = ? AND user_id = ? AND deleted_at IS NULL", columnIDStr, userPayload.ID).Find(&punches).Error; err != nil {
		log.Error("查询打卡记录失败", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	todayPunchCount := 0
	// 今日是否已打卡（使用北京时间）
	today := getTodayStart()
	hasPunchedToday := false
	for _, punch := range punches {
		if punch.CreatedAt.After(today) || punch.CreatedAt.Equal(today) {
			hasPunchedToday = true
			todayPunchCount += 1
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
	database.DB.Model(&model.Punch{}).Where("column_id = ? AND user_id = ? ", columnIDStr, userPayload.ID).Count(&myCount)

	response.Success(c, gin.H{
		"records":           result,
		"user_count":        userCount,
		"my_count":          myCount,
		"punched_today":     hasPunchedToday,
		"today_punch_count": todayPunchCount,
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

	var punch model.Punch
	if err := database.DB.First(&punch, "id = ? AND user_id = ?", punchID, userPayload.ID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("打卡记录不存在或无权限"))
		return
	}

	// 审核通过的打卡不允许删除
	if punch.Status == 1 {
		response.Fail(c, response.ErrInvalidRequest.WithTips("审核通过的打卡记录不允许删除"))
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
	Content  string   `json:"content" binding:"required,max=1000"`
	Images   []string `json:"images" binding:"omitempty,max=9"`
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
	var req PunchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	var punch model.Punch
	if err := database.DB.First(&punch, "id = ? AND user_id = ?", idStr, userPayload.ID).Error; err != nil {
		response.Fail(c, response.ErrNotFound.WithTips("打卡记录不存在或无权限"))
		return
	}

	// 审核通过的打卡不允许修改
	if punch.Status == 1 {
		response.Fail(c, response.ErrInvalidRequest.WithTips("审核通过的打卡记录不允许修改"))
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
type PunchWithImgsAndUser struct {
	Punch    model.Punch `json:"punch"`
	Imgs     []string    `json:"imgs"`
	NickName string      `json:"nick_name"`
	Stared   bool        `json:"stared"`
}

func GetPendingPunchList(c *gin.Context) {
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
	// 只允许管理员或有权限的用户查看
	if userPayload.RoleID < 1 { // 假设1为审核权限
		response.Fail(c, response.ErrForbidden)
		return
	}

	columnIDStr := c.Query("column_id")
	var punches []model.Punch
	query := database.DB.Where("status = 0")
	if columnIDStr != "" {
		query = query.Where("column_id = ?", columnIDStr)
	}
	if err := query.Order("created_at desc").Find(&punches).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	var result []PunchWithImgsAndUser
	for _, punch := range punches {
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}

		var user model.User
		database.DB.Select("nick_name").First(&user, "id = ?", punch.UserID)

		// 查询是否被收藏
		var starCount int64
		database.DB.Model(&model.Star{}).Where("punch_id = ? AND user_id = ?", punch.ID, userPayload.ID).Count(&starCount)
		stared := starCount > 0

		result = append(result, PunchWithImgsAndUser{
			Punch:    punch,
			Imgs:     imgUrls,
			NickName: user.NickName,
			Stared:   stared,
		})
	}
	response.Success(c, struct {
		Total int                    `json:"total"`
		Ps    []PunchWithImgsAndUser `json:"punches"`
	}{
		Total: len(punches),
		Ps:    result,
	})
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
	var punches []model.Punch
	if err := database.DB.Where("user_id = ?", userPayload.ID).Order("created_at desc").Find(&punches).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	type MyPunchWithInfo struct {
		Punch        model.Punch `json:"punch"`
		Imgs         []string    `json:"imgs"`
		ColumnName   string      `json:"column_name"`
		ProjectName  string      `json:"project_name"`
		ActivityName string      `json:"activity_name"`
	}

	var result []MyPunchWithInfo
	for _, punch := range punches {
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}
		var col model.Column
		var colName, projName, actName string
		if err := database.DB.First(&col, "id = ?", punch.ColumnID).Error; err == nil {
			colName = col.Name
			if col.ProjectID != 0 {
				var proj model.Project
				if err := database.DB.First(&proj, "id = ?", col.ProjectID).Error; err == nil {
					projName = proj.Name
					if proj.ActivityID != 0 {
						var act model.Activity
						if err := database.DB.First(&act, "id = ?", proj.ActivityID).Error; err == nil {
							actName = act.Name
						}
					}
				}
			}
		}
		result = append(result, MyPunchWithInfo{
			Punch:        punch,
			Imgs:         imgUrls,
			ColumnName:   colName,
			ProjectName:  projName,
			ActivityName: actName,
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
	var punches []model.Punch
	database.DB.Where("user_id = ?", userPayload.ID).Order("created_at desc").Find(&punches)

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

func GetTodayPunchCount(c *gin.Context) {
	columnId := c.Param("column_id")
	if columnId == "" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("栏目ID不能为空"))
		return
	}
	var count int64
	today := getTodayStart() // 北京时间今日零点
	if err := database.DB.Model(&model.Punch{}).Where("column_id = ? AND created_at >= ?", columnId, today).Count(&count).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	response.Success(c, gin.H{"today_punch_count": count})
}

type PunchWithColumn struct {
	model.Punch
	JoinColumnID  sql.NullInt64 `gorm:"column:join_column_id"`  // 用于判断栏目是否存在
	ColumnOwnerID sql.NullInt64 `gorm:"column:column_owner_id"` // 用于权限判断
}

func GetPunchDetail(c *gin.Context) {
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
	studentID := userPayload.ID
	punchID := c.Param("id")

	if punchID == "" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("打卡ID不能为空"))
		return
	}

	var pc PunchWithColumn
	err := database.DB.
		Table("punch AS p").
		Select("p.*, c.id AS join_column_id, c.owner_id AS column_owner_id").
		Joins("LEFT JOIN `column` c ON c.id = p.column_id").
		Where("p.id = ?", punchID).
		Take(&pc).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// punches 里没有这条记录
		response.Fail(c, response.ErrNotFound.WithTips("打卡记录不存在"))
		return
	}
	if err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 栏目是否存在（LEFT JOIN 成功但 c.id 为空 => 栏目不存在/被删）
	if !pc.JoinColumnID.Valid {
		response.Fail(c, response.ErrNotFound.WithTips("栏目不存在"))
		return
	}

	// 权限判断：本人、管理员(RoleID>=1)、栏目所有者 三者之一即可
	isOwner := pc.UserID == studentID
	isAdmin := userPayload.RoleID >= 1
	isColumnOwner := pc.ColumnOwnerID.Valid && uint(pc.ColumnOwnerID.Int64) == studentID

	if !(isOwner || isAdmin || isColumnOwner) {
		response.Fail(c, response.ErrForbidden)
		return
	}

	var imgs []model.PunchImg
	database.DB.Where("punch_id = ?", punchID).Find(&imgs)
	imgUrls := make([]string, 0, len(imgs))
	for _, img := range imgs {
		imgUrls = append(imgUrls, img.ImgURL)
	}

	var stars []model.Star
	err = database.DB.Where("punch_id = ? AND user_id = ?", punchID, studentID).Find(&stars).Error

	var stared = false
	if len(stars) > 0 {
		stared = true
	}

	response.Success(c, gin.H{
		"punch":  pc.Punch,
		"stared": stared,
		"imgs":   imgUrls,
	})
}

// 获取已审核的打卡列表
func GetReviewedPunchList(c *gin.Context) {
	// 获取认证信息并验证权限
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
	// 只允许管理员或有权限的用户查看
	if userPayload.RoleID < 1 {
		response.Fail(c, response.ErrForbidden)
		return
	}

	// 获取查询参数
	columnIDStr := c.Query("column_id")
	statusStr := c.Query("status") // 可选参数：1-通过, 2-拒绝

	// 构建查询
	query := database.DB.Where("status != 0") // 排除待审核
	if columnIDStr != "" {
		query = query.Where("column_id = ?", columnIDStr)
	}
	if statusStr != "" {
		status, err := strconv.Atoi(statusStr)
		if err == nil && (status == 1 || status == 2) {
			query = query.Where("status = ?", status)
		}
	}

	// 查询打卡记录
	var punches []model.Punch
	if err := query.Order("created_at desc").Find(&punches).Error; err != nil {
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 组装返回数据
	var result []PunchWithImgsAndUser
	for _, punch := range punches {
		// 查询打卡图片
		var imgs []model.PunchImg
		database.DB.Where("punch_id = ?", punch.ID).Find(&imgs)
		imgUrls := make([]string, 0, len(imgs))
		for _, img := range imgs {
			imgUrls = append(imgUrls, img.ImgURL)
		}

		// 查询用户昵称
		var user model.User
		database.DB.Select("nick_name").First(&user, "id = ?", punch.UserID)
		var exist bool
		err := database.DB.
			Raw("SELECT EXISTS(SELECT 1 FROM star WHERE user_id = ? AND punch_id = ? )",
				userPayload.ID, punch.ID).
			Scan(&exist).Error
		if err != nil {
			response.Fail(c, response.ErrDatabase)
			return
		}
		result = append(result, PunchWithImgsAndUser{
			Punch:    punch,
			Imgs:     imgUrls,
			NickName: user.NickName,
			Stared:   exist,
		})
	}

	response.Success(c, struct {
		Total int                    `json:"total"`
		Ps    []PunchWithImgsAndUser `json:"punches"`
	}{
		Total: len(punches),
		Ps:    result,
	})
	return
}

// PresignedUploadRequest 预签名上传请求
type PresignedUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type"`
}

// GetPresignedUploadURL 获取预签名上传 URL
func GetPresignedUploadURL(c *gin.Context) {
	// 获取认证信息
	payload, exists := c.Get("payload")
	if !exists {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	_, ok := payload.(*jwt.Claims)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}

	// 绑定请求参数
	var req PresignedUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定预签名上传请求失败", "error", err)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 创建图片床实例
	pb := pictureBed.NewPictureBed(config.Get().S3.Endpoint, "")

	// 初始化 S3 客户端
	if err := pb.InitS3(c.Request.Context()); err != nil {
		log.Error("初始化 S3 客户端失败", "error", err)
		response.Fail(c, response.ErrServerInternal.WithTips("初始化存储服务失败"))
		return
	}

	// 生成预签名上传 URL
	presignedReq := pictureBed.PresignedUploadRequest{
		Filename:    req.Filename,
		ContentType: req.ContentType,
		ExpiresIn:   120, // 2 分钟
	}

	presignedResp, err := pb.GeneratePresignedUploadURL(c.Request.Context(), presignedReq)
	if err != nil {
		log.Error("生成预签名上传 URL 失败", "error", err)
		response.Fail(c, response.ErrServerInternal.WithTips("生成上传链接失败"))
		return
	}

	response.Success(c, presignedResp)
}
