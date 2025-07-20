package model

type Project struct {
	Model
	Name        string   `gorm:"type:varchar(100);not null" json:"name"`    // 项目名称
	Description string   `gorm:"type:varchar(255);" json:"description"`     // 项目描述
	OwnerID     string   `gorm:"type:varchar(20);not null" json:"owner_id"` // 所有者学号，外键指向用户表的学号
	ActivityID  uint     `gorm:"default:null" json:"activity_id"`           // 关联的活动ID（暂时不设外键约束）
	Activity    Activity `gorm:"-" json:"activity,omitempty"`               // 关联到活动模型（手动关联）
	StartDate   int64    `gorm:"" json:"start_date"`                        // 项目开始时间
	EndDate     int64    `gorm:"" json:"end_date"`                          // 项目结束时间
	Avatar      string   `gorm:"type:varchar(255);" json:"avatar"`          // 项目封面URL
	// 关联到用户
	User User `gorm:"foreignKey:OwnerID;references:StudentID" json:"user"` // 关联到用户模型，使用学号作为外键
}
