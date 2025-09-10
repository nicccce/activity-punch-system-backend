package model

type Project struct {
	Model
	Name        string   `gorm:"type:varchar(100);not null" json:"name" excel:"项目名称"`           // 项目名称
	Description string   `gorm:"type:varchar(255);" json:"description" excel:"项目描述"`            // 项目描述
	OwnerID     string   `gorm:"type:varchar(20);not null" json:"owner_id" excel:"所有者学号/工号"`    // 所有者学号，外键指向用户表的学号
	ActivityID  uint     `gorm:"default:null" json:"activity_id" excel:"-"`                     // 关联的活动ID
	Activity    Activity `gorm:"foreignKey:ActivityID;references:ID" json:"activity" excel:"-"` // 关联到活动模型
	StartDate   int64    `gorm:"" json:"start_date" excel:"项目开始时间"`                             // 项目开始时间
	EndDate     int64    `gorm:"" json:"end_date" excel:"项目结束时间"`                               // 项目结束时间
	Avatar      string   `gorm:"type:varchar(255);" json:"avatar" excel:"项目封面URL"`              // 项目封面URL
	// 关联到用户
	User User `gorm:"foreignKey:OwnerID;references:StudentID" json:"user" excel:"-"` // 关联到用户模型，使用学号作为外键
}
