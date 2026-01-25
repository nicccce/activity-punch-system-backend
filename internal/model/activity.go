package model

type Activity struct {
	Model
	Name            string `gorm:"type:varchar(100);not null" json:"name" `   // 活动名称
	Description     string `gorm:"type:varchar(255);" json:"description" `    // 活动描述
	OwnerID         string `gorm:"type:varchar(20);not null" json:"owner_id"` // 所有者学号，外键指向用户表的学号
	StartDate       int64  `gorm:"" json:"start_date"`                        // 活动开始时间
	EndDate         int64  `gorm:"" json:"end_date"`                          // 活动结束时间
	Avatar          string `gorm:"type:varchar(255);" json:"avatar"`          // 活动封面URL
	DailyPointLimit uint   `gorm:"default:0" json:"daily_point_limit"`        // 每日积分上限，0表示不限制
	CompletionBonus uint   `gorm:"default:0" json:"completion_bonus"`         // 完成活动所有栏目后的额外奖励积分，0表示无奖励
	// 关联到用户
	User User `gorm:"foreignKey:OwnerID;references:StudentID" json:"user"` // 关联到用户模型，使用学号作为外键
}
