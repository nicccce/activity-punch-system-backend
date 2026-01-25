package model

type Column struct {
	Model
	Name            string  `gorm:"type:varchar(100);not null" json:"name" excel:"栏目名称"`          // 栏目名称
	Description     string  `gorm:"type:varchar(255);" json:"description" excel:"栏目描述"`           // 栏目描述
	OwnerID         string  `gorm:"type:varchar(20);not null" json:"owner_id" excel:"所有者学号/工号"`   // 所有者学号，外键指向用户表的学号
	ProjectID       uint    `gorm:"default:null" json:"project_id" excel:"-"`                     // 关联的项目ID
	Project         Project `gorm:"foreignKey:ProjectID;references:ID" json:"project" excel:"-"`  // 关联到项目模型
	StartDate       int64   `gorm:"" json:"start_date" excel:"栏目开始时间"`                            // 栏目开始时间
	EndDate         int64   `gorm:"" json:"end_date" excel:"栏目结束时间"`                              // 栏目结束时间
	Avatar          string  `gorm:"type:varchar(255);" json:"avatar" excel:"栏目封面URL"`             // 栏目封面URL
	DailyPunchLimit int     `gorm:"default:0;not null" json:"daily_punch_limit" excel:"每日可打卡次数"`  // 每日可打卡次数，0表示不限次数
	PointEarned     uint    `gorm:"default:0;not null" json:"point_earned" excel:"每次打卡可获得的积分"`    // 每次打卡可获得的积分
	StartTime       string  `gorm:"type:varchar(10);not null" json:"start_time" excel:"每日打卡开始时间"` // 每日打卡开始时间，格式为 "HH:MM"
	EndTime         string  `gorm:"type:varchar(10);not null" json:"end_time" excel:"每日打卡结束时间"`   // 每日打卡结束时间，格式为 "HH:MM"
	Optional        bool    `gorm:"default:false" json:"optional" excel:"特殊栏目"`                   // 特殊栏目，不计入完成所有栏目的判断
	// 关联到用户
	User User `gorm:"foreignKey:OwnerID;references:StudentID" json:"user" excel:"-"` // 关联到用户模型，使用学号作为外键
}
