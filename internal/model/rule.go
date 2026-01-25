package model

import (
	"gorm.io/gorm"
	"strconv"
	"time"
)
import "gorm.io/datatypes"

type Rule struct {
	ID         uint `gorm:"primaryKey" excel:"id"`
	ActivityID uint `gorm:"column:activity_id;not null" json:"-"`
}
type IntegrityRule struct {
	Rule
	ColumnIDs   datatypes.JSONSlice[uint] `gorm:"type:json;column:column_id;not null"json:"column_ids"`
	PointEarned uint                      `gorm:"column:point_earned;not null" json:"point_earned" `
}
type ContinuityRule struct {
	Rule
	Day         uint                      `gorm:"column:day;not null" json:"day"`
	PointEarned uint                      `gorm:"column:point_earned;not null" json:"point_earned" `
	Once        bool                      `gorm:"column:once;default:true" json:"once"`
	ColumnIDs   datatypes.JSONSlice[uint] `gorm:"type:json;column:column_id;not null"json:"column_ids"`
}

func (r *Rule) Execute(tx *gorm.DB, trigger *Punch) (addScore uint, err error) {
	var integrityRules []IntegrityRule
	var as uint
	addScore = 0
	t := &trigger.CreatedAt
	start := time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 0, 0,
		t.Location(),
	)
	end := start.Add(24 * time.Hour)
	err = nil
	if err = tx.Model(&IntegrityRule{}).Where("activity_id=?", r.ActivityID).Find(integrityRules).Error; err != nil {
		return
	}
	for _, ir := range integrityRules {
		as, err = ir.Execute(tx, trigger, &start, &end)
		if err != nil {
			return
		}
		addScore += as
	}

	var continuityRules []ContinuityRule
	if err = tx.Model(&ContinuityRule{}).Where("activity_id=?", r.ActivityID).Find(continuityRules).Error; err != nil {
		return
	}
	for _, cr := range continuityRules {
		as, err = cr.Execute(tx, trigger)
		if err != nil {
			return
		}
		addScore += as
	}
	return addScore, nil
}

func (r *IntegrityRule) Execute(tx *gorm.DB, punch *Punch, start *time.Time, end *time.Time) (addScore uint, err error) {
	var columnCount int64
	if err = tx.Model(&Punch{}).Where("created_at >= ? AND created_at < ? AND status=1 AND  column_id IN (?)", start, end, r.ColumnIDs).Distinct("column_id").Count(&columnCount).Error; err != nil {
		return
	}
	if columnCount == int64(len(r.ColumnIDs)) {
		score := Score{
			UserID:   punch.UserID,
			Count:    r.PointEarned,
			Cause:    "IntegrityRule" + strconv.FormatUint(uint64(r.ID), 10),
			MarkedBy: "system",
			PunchID:  punch.ID,
			ColumnID: 0,
		}
		if err = tx.Create(&score).Error; err != nil {
			return
		}
		return r.PointEarned, nil
	}
	return 0, nil
}
func (r *ContinuityRule) Execute(tx *gorm.DB, punch *Punch) (addScore uint, err error) {
	if r.Once {
		var exist bool
		if err = tx.Raw("SELECT EXISTS(SELECT 1 FROM score WHERE user_id = ?  AND deleted_at IS NULL AND cause = ? AND marked_by = ?  )",
			punch.UserID, "ContinuityRule"+strconv.FormatUint(uint64(r.ID), 10), "system").Scan(&exist).Error; err != nil {
			return
		}
		if exist {
			return 0, nil
		}
	}
	var ccs []uint
	if err = tx.Model(&ColumnContinuity{}).Select("current").Where("column_id IN (?) AND user_id = ?", r.ColumnIDs, punch.UserID).Find(&ccs).Error; err != nil {
		return
	}
	for _, cc := range ccs {
		if cc < r.Day {
			return 0, nil
		}
	}
	score := Score{
		UserID:   punch.UserID,
		Count:    r.PointEarned,
		Cause:    "ContinuityRule" + strconv.FormatUint(uint64(r.ID), 10),
		MarkedBy: "system",
		PunchID:  punch.ID,
		ColumnID: 0,
	}
	if err = tx.Create(&score).Error; err != nil {
		return
	}
	return r.PointEarned, nil
}
