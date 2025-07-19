package model

type Project struct {
	Model
	Name        string `gorm:"type:varchar(100);not null"` // 项目名称
	Description string `gorm:"type:varchar(255);"`         // 项目描述
	OwnerID     string `gorm:"type:varchar(20);not null"`  // 所有者学号，外键指向用户表的学号
	StartDate   int64  `gorm:""`                           // 项目开始时间
	EndDate     int64  `gorm:""`                           // 项目结束时间
	Category    string `gorm:"type:varchar(50);not null"`  // 项目类别（如暑期打卡、寒假打卡等）
	// 关联到用户
	User User `gorm:"foreignKey:OwnerID;references:StudentID"` // 关联到用户模型，使用学号作为外键
}
