package model

import (
	"gorm.io/gorm"
)

type Score struct {
	Model
	//我想还是uint好吧，前端自己做"汇率转换"
	Count    uint   `gorm:"not null" json:"count"`
	UserID   uint   `gorm:"not null" json:"-"`
	MarkedBy string `gorm:"type:varchar(50);not null" json:"marked_by"`
	Cause    string `gorm:"type:varchar(255);not null" json:"cause"`
	PunchID  uint   `gorm:"not null" json:"-"`
	ColumnID uint   `gorm:"not null" json:"-"`

	Punch partialPunchForScore `gorm:"foreignKey:PunchID;references:ID"`
	//不该这样的，但这样把ColumnID也放在了表里很方便应该是不负责打分部分的NIA_sai做强制"实时求和"统计(这种玩意有必要写吗？性能差不说，ACID的A是拿来看的吗？
	Column partialColumnForScore `gorm:"foreignKey:ColumnID;references:ID" json:"column"`
}

func (s *Score) AfterCreate(tx *gorm.DB) (err error) {
	return afterScoreChange(s, tx, true)
}
func (s *Score) AfterDelete(tx *gorm.DB) (err error) {
	return afterScoreChange(s, tx, false)
}

func afterScoreChange(s *Score, tx *gorm.DB, sign bool) (err error) {
	var t TotalScore
	if err = tx.Model(&TotalScore{}).Where("activity_id = ? AND user_id = ?",
		tx.Statement.Context.Value("activity_id"), //todo: 记得加进context,或者自己写
		s.UserID).Find(&t).Error; err == nil {
		if sign {
			t.Score += s.Count
		} else {
			t.Score -= s.Count
		}
		if err = tx.Save(&t).Error; err != nil {
			return
		}
	}
	return
}

type partialPunchForScore struct {
	Model
	//Content  string `gorm:"type:varchar(255);not null" json:"content"`
}

// 也写着，以免之后前端提出不一样的信息需求
type partialColumnForScore struct {
	ID        uint                   `gorm:"primaryKey" json:"id"`
	Project   partialProjectForScore `gorm:"foreignKey:ProjectID;references:ID" json:"project"`
	ProjectID uint                   `gorm:"not null" json:"-"`
	Name      string                 `gorm:"type:varchar(100);not null" json:"name"`
}
type partialProjectForScore struct {
	ID uint `gorm:"primaryKey" json:"id"`
	//Activity   partialActivityForScore `gorm:"foreignKey:ActivityID;references:ID" json:"activity"`
	//ActivityID uint                    `gorm:"not null" json:"-"`
	Name string `gorm:"type:varchar(100);not null" json:"name"`
}
type partialActivityForScore struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"type:varchar(100);not null" json:"name"`
}

func (partialPunchForScore) TableName() string {
	return "punch"
}
func (partialColumnForScore) TableName() string {
	return "column"
}
func (partialProjectForScore) TableName() string {
	return "project"
}
func (partialActivityForScore) TableName() string {
	return "activity"
}
