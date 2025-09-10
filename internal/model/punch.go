package model

import (
	"gorm.io/gorm"
)

type Punch struct {
	Model
	//ID       int    `gorm:"primaryKey" json:"id"`
	ColumnID int    `gorm:"not null" json:"column_id"`
	UserID   string `gorm:"not null" json:"user_id"`
	Content  string `gorm:"type:varchar(255);not null" json:"content"`
	Status   int    `gorm:"not null" json:"status"` //status为  0 待审核   1 审核通过   2 不通过
}

// todo: 未测 打卡能被删除吗？
func (p *Punch) AfterCreate(tx *gorm.DB) (err error) {
	var c Continuity
	if err = tx.Model(&Continuity{}).Where("activity_id = ? AND user_id = ?",
		tx.Statement.Context.Value("activity_id"), //todo: 记得加进context,或者自己写
		p.UserID).Find(&c).Error; err == nil {
		c.RefreshTo(p.CreatedAt)
		if err = tx.Save(&c).Error; err != nil {
			return
		}
	}
	return
}
