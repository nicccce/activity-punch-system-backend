package model

import "time"

type Star struct {
	//Type  int //区别收藏类型
	UserID    uint      `gorm:"type:varchar(20);not null;index:idx_user_punch,unique" json:"-"`
	PunchID   uint      `gorm:"not null;index:idx_user_punch,unique" json:"-"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	//甚为丑陋
	Punch partialPunchForStar `gorm:"foreignKey:PunchID;references:ID" json:"punch"` //额外键关联处字段名是代码原字段名ID和id谔谔

}
type partialPunchForStar struct {
	Model
	Column   partialColumnForStar `gorm:"foreignKey:ColumnID;references:ID" json:"column"`
	ColumnID uint                 `gorm:"not null" json:"-"`
	UserID   string               `gorm:"not null" json:"user_id"`
	Content  string               `gorm:"type:varchar(255);not null" json:"content"`
	Status   int                  `gorm:"not null" json:"status"` //status为  0 待审核   1 审核通过   2 不通过
}
type partialColumnForStar struct {
	ID        int                   `gorm:"primaryKey" json:"id"`
	Project   partialProjectForStar `gorm:"foreignKey:ProjectID;references:ID" json:"project"`
	ProjectID uint                  `gorm:"not null" json:"-"`
	Name      string                `gorm:"type:varchar(100);not null" json:"name"`
}
type partialProjectForStar struct {
	ID         int                    `gorm:"primaryKey" json:"id"`
	Activity   partialActivityForStar `gorm:"foreignKey:ActivityID;references:ID" json:"activity"`
	ActivityID uint                   `gorm:"not null" json:"-"`
	Name       string                 `gorm:"type:varchar(100);not null" json:"name"`
}
type partialActivityForStar struct {
	ID   int    `gorm:"primaryKey" json:"id"`
	Name string `gorm:"type:varchar(100);not null" json:"name"`
}

func (partialPunchForStar) TableName() string {
	return "punch"
}
func (partialColumnForStar) TableName() string {
	return "column"
}
func (partialProjectForStar) TableName() string {
	return "project"
}
func (partialActivityForStar) TableName() string {
	return "activity"
}
