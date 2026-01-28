package activity

import (
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/model"
)

func selectHistory(userId uint, askTime int64, offset, limit int, r *[]model.Activity) error {
	subQuery := database.DB.
		Table("punch").
		Select("MAX(create_at) as last_time, activity_id").
		Where("user_id = ? AND create_at <= ?", userId, askTime).
		Group("activity_id")
	return database.DB.
		Table("(?) as recent", subQuery).
		Select("activity.*").
		Joins("JOIN activity ON activity.id = recent.activity_id").
		Order("recent.last_time DESC").
		Offset(offset).
		Limit(limit).
		Scan(r).Error
}
func getColumnIds(id uint) (columnIDs []uint, err error) {
	err = database.DB.Table("column").
		Select("column.id").
		Joins("JOIN project ON project.id = column.project_id").
		Where("project.activity_id = ?", id).
		Pluck("column.id", &columnIDs).Error
	if err != nil {
		log.Error("数据库 通过activity id获取column ids失败", "error", err.Error())
		return nil, err
	}
	return columnIDs, nil
}

type rank struct {
	Rank uint `gorm:"column:ranks" json:"rank" `
	model.TotalScore
}

func selectRank(activityID uint, offset, limit int) ([]rank, int64, error) {
	var ranks []rank
	var total int64
	wrapper := database.DB.Model(&model.TotalScore{}).Where("activity_id = ?", activityID)
	if err := wrapper.Count(&total).Error; err != nil {
		log.Error("数据库 查询活动排名失败", "error", err.Error())
		return nil, 0, err
	}
	if err := wrapper.
		Select(`
            user_id,
            score,
            RANK() OVER (ORDER BY score DESC) AS ranks
        `).
		Preload("User").
		Order("ranks ASC").
		Limit(limit).
		Offset(offset).
		Find(&ranks).Error; err != nil {
		log.Error("数据库 查询活动排名失败", "error", err.Error())
		return nil, 0, err
	}
	return ranks, total, nil
}

type activityRankInExcel struct {
	Rank      uint   `gorm:"column:ranks" json:"rank" excel:"排名"`
	Name      string `gorm:"column:name" json:"name" excel:"姓名"`
	NickName  string `gorm:"column:nick_name" json:"nick_name" excel:"昵称"`
	Score     uint   `gorm:"not null" json:"score" excel:"分数"`
	StudentID string `gorm:"column:student_id" json:"student_id" excel:"学号"`
	ID        uint   `gorm:"column:id" json:"user_id" excel:"用户ID"`
	College   string `gorm:"column:college" json:"college" excel:"学院"`
	Major     string `gorm:"column:major" json:"major" excel:"专业"`
	Grade     string `gorm:"column:grade" json:"grade" excel:"年级"`
}

func selectActivityRankInExcel(activityID uint) ([]activityRankInExcel, error) {
	var ranks []activityRankInExcel
	if err := database.DB.Table("total_score ts").
		Select(`
			u.id,
        	u.student_id,
        	u.name,
        	u.nick_name,
        	u.college,
        	u.major,
			u.grade,
			ts.score,
            RANK() OVER (ORDER BY ts.score DESC) AS ranks
        `).
		Joins("JOIN user u ON u.id = ts.user_id").
		Order("ranks ASC").
		Where("ts.activity_id = ?", activityID).
		Scan(&ranks).Error; err != nil {
		return nil, err
	}
	return ranks, nil
}

type briefResult struct {
	Rank              int  `gorm:"column:ranks" json:"rank"`
	TodayPuncherCount uint `json:"today_punched_user_count"`
	TotalScore        uint `gorm:"column:ts" json:"total_score"`
	model.Continuity
}

func briefStats(activityID, userID uint, columnIDs []uint, askTime int64, result *briefResult) error {
	var continuityResult model.Continuity
	if err := database.DB.Table("continuity").Where("activity_id = ? AND user_id = ?", activityID, userID).
		Scan(&continuityResult).Error; err != nil {
		log.Error("数据库 查询continuity失败", "error", err.Error())
		return err
	}

	var totalScoreResult briefResult
	subQuery := database.DB.Table("total_score").
		Select("user_id, score AS ts, RANK() OVER (ORDER BY score DESC) AS ranks").
		Where("activity_id = ?", activityID)
	if err := database.DB.Table("(?) AS ranked", subQuery).
		Where("user_id = ?", userID).
		Scan(&totalScoreResult).Error; err != nil {
		log.Error("数据库 查询total_score失败", "error", err.Error())
		return err
	}
	var todayPuncherCount uint
	if err := database.DB.Table("punch").
		Select("COUNT(DISTINCT user_id) AS tpuc").
		Where("column_id IN (?) AND created_at >= ?", columnIDs, askTime-askTime%86400). //就不再created_at<=asktime了
		Scan(&todayPuncherCount).Error; err != nil {
		log.Error("数据库 查询punch获得当天已经打卡此活动人数失败", "error", err.Error())
		return err
	}
	result.Continuity = continuityResult
	result.TotalScore = totalScoreResult.TotalScore
	result.Rank = totalScoreResult.Rank
	result.TodayPuncherCount = todayPuncherCount
	return nil
}
