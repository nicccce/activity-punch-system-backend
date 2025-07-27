package model

import "time"

type Star struct {
	//Type  int //区别收藏类型
	UserID    string    `gorm:"type:varchar(20);not null;index:idx_user_punch,unique" json:"-"`
	PunchID   uint      `gorm:"not null;index:idx_user_punch,unique" json:"-"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	Punch     Punch     `gorm:"foreignKey:PunchID;references:ID" json:"punch"` //额外键关联处字段名是代码原字段名ID和id谔谔
}
