package model

import "gorm.io/gorm"
import "gorm.io/datatypes"

type Rule struct {
	ActivityID uint
	trigger    Punch
}
type IntegrityRule struct {
	ActivityID  uint                      `gorm:"column:activity_id;not null" json:"-"`
	ColumnIDs   datatypes.JSONSlice[uint] `gorm:"type:json;column:column_id;not null"json:"column_ids"`
	PointEarned uint                      `gorm:"column:point_earned;not null" json:"point_earned" `
}
type ContinuityRule struct {
	ActivityID  uint `gorm:"column:activity_id;not null" json:"-"`
	Day         uint `gorm:"column:day;not null" json:"day"`
	PointEarned uint `gorm:"column:point_earned;not null" json:"point_earned" `
	ColumnID    uint `gorm:"column:column_id;not null" json:"column_id"`
}

func (r *Rule) Execute(tx *gorm.DB) (err error) {
	var integrityRules []IntegrityRule
	if err = tx.Model(&IntegrityRule{}).Where("activity_id=?", r.ActivityID).Find(integrityRules).Error; err != nil {
		return
	}
	for _, ir := range integrityRules {
		if err = ir.Execute(tx, r.trigger.CreatedAt); err != nil {
			return
		}
	}

	var continuityRules []ContinuityRule
	if err = tx.Model(&ContinuityRule{}).Where("activity_id=?", r.ActivityID).Find(continuityRules).Error; err != nil {
		return
	}
	for _, cr := range continuityRules {
		if err = cr.Execute(tx); err != nil {
			return
		}
	}
	return nil
}

func (r *IntegrityRule) Execute(tx *gorm.DB) (err error) {
	return
}
func (r *ContinuityRule) Execute(tx *gorm.DB) (err error) {
	return
}
