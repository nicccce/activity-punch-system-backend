package model

import (
	"gorm.io/gorm"
)

type Punch struct {
	Model
	//ID       int    `gorm:"primaryKey" json:"id"`
	ColumnID int    `gorm:"not null" json:"column_id"`
	UserID   uint   `gorm:"not null" json:"user_id"`
	Content  string `gorm:"type:varchar(255);not null" json:"content"`
	Status   int    `gorm:"not null" json:"status"` //status为  0 待审核   1 审核通过   2 不通过
}

// todo: 打卡能被删除吗？
func (p *Punch) AfterCreate(tx *gorm.DB) (err error) {
	c := Continuity{FkUserActivity: *(tx.Statement.Context.Value("fk_user_activity").(*FkUserActivity))}
	if err = tx.Model(&Continuity{}).
		Where("activity_id = ? AND user_id = ?", c.ActivityID, c.UserID).
		Find(&c).Error; err == nil {
		flag := c.Total
		c.RefreshTo(p.CreatedAt)
		if flag != 0 || tx.Create(&c).Error != nil {
			return tx.Model(&Continuity{}).Where("activity_id = ? AND user_id = ?", c.ActivityID, c.UserID).Updates(c).Error
		}
	}
	return
}
