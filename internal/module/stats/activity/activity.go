package activity

import (
	"activity-punch-system/internal/global/context"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/logger"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"activity-punch-system/internal/module/stats/tool"
	"activity-punch-system/internal/module/stats/tree"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var log = logger.New("Stats-Activity")

// History 获取活动历史
func History(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	offset, limit := tool.GetPage(c)
	askTime := tool.GetTime(c)
	var result []model.Activity
	err := selectHistory(user.ID, askTime, offset, limit, &result)
	if err != nil {
		log.Error("数据库 查询 activity 表错误", "error", err)
		response.Fail(c, response.ErrDatabase)
		return
	}
	response.Success(c, result)
}
func Rank(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	a, ok := activityIdValidator(c)
	if !ok {
		return
	}
	offset, limit := tool.GetPage(c)
	forceStr := c.Query("force")
	force := false
	if forceStr == "true" {
		force = true
	}

	//强制遍历一遍来更新
	if force && user.RoleID > 0 { //todo: 权限对吗
		columnIds, err := getColumnIds(a.ID)
		if err != nil {
			response.Fail(c, response.ErrDatabase)
			return
		}
		totalScores := make(map[uint]uint)

		var scores []model.Score
		//多次sql获取各column的score记录再累加
		for _, columnId := range columnIds {
			if err = database.DB.Model(&model.Score{}).
				Where("column_id = ? AND deleted_at IS NULL", columnId).
				Find(&scores).Error; err != nil {
				log.Error("数据库 通过column id查询 score 表错误", "error", err.Error())
				response.Fail(c, response.ErrDatabase)
				return
			}
			for _, score := range scores {
				totalScores[score.UserID] += score.Count
			}
		}
		//or  一次sql获取所有column的score记录再累加 (也许效率更高但score太多的话我看未必好,不如考验一下mysql
		//if err := database.DB.Model(&model.Score{}).
		//	Where("column_id IN ? AND deleted_at IS NULL", columnIds).
		//	Find(&scores).Error; err != nil {
		//	log.Error("数据库查询 score 表错误", "error", err.Error())
		//	response.Fail(c, response.ErrDatabase)
		//	return
		//} else {
		//	for _, score := range scores {
		//		totalScores[score.UserID] += score.Count
		//	}
		//哈哈 所以就不该写这个
		for id, score := range totalScores {
			if err = database.DB.Table("total_score").Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "activity_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"score"}),
			}).Create(&model.TotalScore{
				UserID:     id,
				ActivityID: a.ID,
				Score:      score,
			}).Error; err != nil {
				log.Error("数据库 更新 total_score 表错误", "error", err.Error())
			}

		}
	}
	result, err := selectRank(a.ID, offset, limit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.ErrNotFound)
			return
		}
		response.Fail(c, response.ErrDatabase)
		return
	}
	response.Success(c, result)
}

func Detail(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	a, ok := activityIdValidator(c)
	if !ok {
		return
	}
	offset, limit := tool.GetPage(c)
	columnIDs, err := getColumnIds(a.ID)
	if err != nil {
		response.Fail(c, response.ErrDatabase)
		return
	}
	var result []model.Score
	if err := database.DB.Model(&model.Score{}).Preload("Punch").Preload("Column").Preload("Column.Project").
		Where("deleted_at IS NULL AND column_id in (?) AND user_id = ?", columnIDs, user.ID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).Find(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.ErrNotFound)
			return
		}
		log.Error("数据库 查询 score 表错误", "error", err.Error())
		response.Fail(c, response.ErrDatabase)
		return
	}
	response.Success(c, result)
}

// Brief 获取某活动的今日()已经打卡人数,
// 请求者参与打卡次数,
// 请求者在该活动下的总得分,
// 以及请求者在该活动下的排名(按打卡总得分排名)
// ....困了
func Brief(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	a, ok := activityIdValidator(c)
	if !ok {
		return
	}
	columnIDs, err := getColumnIds(a.ID)
	if err != nil {
		response.Fail(c, response.ErrDatabase)
		return
	}
	var result briefResult
	if err := briefStats(a.ID, user.ID, columnIDs, time.Now().Unix(), &result); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.ErrNotFound)
			return
		}
		log.Error("查询 column 表错误", "error", err)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	response.Success(c, result)
}
func ExportMyStats2Json(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	askTime := tool.GetTime(c)
	a, ok := activityIdValidator(c)
	if !ok {
		return
	}
	response.Success(c, tree.Unfold3[Activity, Project, Column](&Activity{*a}, user.Id, 0, askTime))
}

func activityIdValidator(c *gin.Context) (*model.Activity, bool) {
	activityId := c.Param("id")
	if activityId == "" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("活动ID不能为空"))
		return nil, false
	}
	var a []model.Activity

	r := database.DB.Model(&model.Activity{}).
		Where("id = ? AND deleted_at IS NULL", activityId).
		Limit(2).Find(&a)
	if r.Error != nil {
		log.Error("查询 activity 表错误", "error", r.Error)
		response.Fail(c, response.ErrDatabase.WithOrigin(r.Error))
		return nil, false
	} else if len(a) == 0 {
		response.Fail(c, response.ErrNotFound)
		return nil, false
	} else if len(a) > 1 {
		log.Warn("查询 activity 表警告", "error", "重复 columnId")
	}
	return &a[0], true
}

type Activity struct{ model.Activity }
type Project struct{ model.Project }
type Column struct{ model.Column }

func (a Activity) GetId() uint {
	return a.ID
}
func (a Activity) GetName() string {
	return a.Name
}
func (a Activity) NextLayer() []tree.Record {
	var ps []Project
	database.DB.Model(&model.Project{}).
		Where("activity_id = ? AND deleted_at IS NULL", a.GetId()).Find(&ps)
	return tree.ToRecordSlice(ps)
}

func (p Project) GetId() uint {
	return p.ID
}
func (p Project) GetName() string {
	return p.Name
}
func (p Project) NextLayer() []tree.Record {
	var cs []Column
	database.DB.Model(&model.Column{}).
		Where("project_id = ? AND deleted_at IS NULL", p.GetId()).Find(&cs)
	return tree.ToRecordSlice(cs)
}
func (c Column) GetId() uint {
	return c.ID
}

func (c Column) GetName() string {
	return c.Name
}

func (c Column) NextLayer() []tree.Record {
	return nil
}
func (c Column) GetScore(userId string, startTime, endTime int64) float64 {
	var rs []tool.Punch
	//todo
	var sum = 0.0
	for _, r := range rs {
		sum += r.Score
	}
	return sum
}
