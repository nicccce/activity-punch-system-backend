package model

import (
	"gorm.io/gorm"
)

type Score struct {
	Model
	//我想还是uint好吧，前端自己做"汇率转换"
	Count    uint   `gorm:"not null" json:"count"  excel:"分数"`
	UserID   uint   `gorm:"not null" json:"-" excel:"-"`
	MarkedBy string `gorm:"type:varchar(50);not null" json:"marked_by" excel:"打分人"`
	Cause    string `gorm:"type:varchar(255);not null" json:"cause" excel:"打分原因"`
	PunchID  uint   `gorm:"not null" json:"-" excel:"-"`
	ColumnID uint   `gorm:"not null" json:"-" excel:"-"`

	Punch partialPunchForScore `gorm:"foreignKey:PunchID;references:ID" excel:"-"`
	//不该这样的，但这样把ColumnID也放在了表里很方便应该是不负责打分部分的NIA_sai做强制"实时求和"统计(这种玩意有必要写吗？性能差不说，ACID的A是拿来看的吗？
	Column partialColumnForScore `gorm:"foreignKey:ColumnID;references:ID" json:"column" excel:"-"`
}

func (s *Score) AfterCreate(tx *gorm.DB) (err error) {
	return afterScoreChange(s, tx, true)
}
func (s *Score) AfterDelete(tx *gorm.DB) (err error) {
	return afterScoreChange(s, tx, false)
}

func afterScoreChange(s *Score, tx *gorm.DB, sign bool) (err error) {
	t := TotalScore{FkUserColumn: *(tx.Statement.Context.Value("fk_user_activity").(*FkUserColumn))}
	if err = tx.Model(&TotalScore{}).Where("activity_id = ? AND user_id = ?",
		t.ColumnID, t.UserID).Find(&t).Error; err == nil {
		flag := t.Score
		if sign {
			t.Score += s.Count
		} else {
			t.Score -= s.Count
		}
		if flag != 0 || tx.Create(&t).Error != nil {
			return tx.Model(&TotalScore{}).Where("activity_id = ? AND user_id = ?", t.ColumnID, t.UserID).Updates(t).Error
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
	//Activity   partialActivityForScore `gorm:"foreignKey:ColumnID;references:ID" json:"activity"`
	//ColumnID uint                    `gorm:"not null" json:"-"`
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
