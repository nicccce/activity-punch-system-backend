package model

import (
	"gorm.io/gorm"
)

type Punch struct {
	Model
	//ID       int    `gorm:"primaryKey" json:"id"`
	ColumnID int    `gorm:"not null" json:"column_id"  excel:"-"`
	UserID   uint   `gorm:"not null" json:"user_id" excel:"-"`
	Content  string `gorm:"type:varchar(255);not null" json:"content" excel:"打卡内容"`
	Status   int    `gorm:"not null" json:"status" excel:"审核状态"` //status为  0 待审核   1 审核通过   2 不通过
}

// todo: 打卡能被删除吗？
func (p *Punch) UpdateContinuity(tx *gorm.DB, activityID, projectID uint) (err error) {
	var (
		cc ColumnContinuity
		pc ProjectContinuity
		ac ActivityContinuity
	)
	day := p.CreatedAt.Unix() / (24 * 60 * 6)
	if err = tx.Model(&ColumnContinuity{}).
		Where("column_id = ? AND user_id = ?", p.ColumnID, p.UserID).
		Find(&cc).Error; err == nil {
		flag := cc.Total
		cc.RefreshTo(day)
		if flag != 0 || tx.Create(&cc).Error != nil {
			if err = tx.Model(&ColumnContinuity{}).Where("column_id = ? AND user_id = ?", cc.ColumnID, cc.UserID).Updates(cc).Error; err != nil {
				return
			}
		}
	}
	if err = tx.Model(&ProjectContinuity{}).
		Where("project_id = ? AND user_id = ?", projectID, p.UserID).
		Find(&pc).Error; err == nil {
		flag := pc.Total
		pc.RefreshTo(day)
		if flag != 0 || tx.Create(&pc).Error != nil {
			if err = tx.Model(&ProjectContinuity{}).Where("project_id = ? AND user_id = ?", pc.ProjectID, pc.UserID).Updates(pc).Error; err != nil {
				return
			}
		}
	}
	if err = tx.Model(&ActivityContinuity{}).
		Where("activity_id = ? AND user_id = ?", activityID, p.UserID).
		Find(&ac).Error; err == nil {
		flag := ac.Total
		ac.RefreshTo(day)
		if flag != 0 || tx.Create(&ac).Error != nil {
			if err = tx.Model(&ActivityContinuity{}).Where("activity_id = ? AND user_id = ?", ac.ActivityID, ac.UserID).Updates(ac).Error; err != nil {
				return
			}
		}
	}
	return
}
