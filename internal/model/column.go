package model

type Column struct {
	Model
	Name        string  `gorm:"type:varchar(100);not null" json:"name"`            // 栏目名称
	Description string  `gorm:"type:varchar(255);" json:"description"`             // 栏目描述
	OwnerID     string  `gorm:"type:varchar(20);not null" json:"owner_id"`         // 所有者学号，外键指向用户表的学号
	ProjectID   uint    `gorm:"default:null" json:"project_id"`                    // 关联的项目ID
	Project     Project `gorm:"foreignKey:ProjectID;references:ID" json:"project"` // 关联到项目模型
	StartDate   int64   `gorm:"" json:"start_date"`                                // 栏目开始时间
	EndDate     int64   `gorm:"" json:"end_date"`                                  // 栏目结束时间
	Avatar      string  `gorm:"type:varchar(255);" json:"avatar"`                  // 栏目封面URL
	// 关联到用户
	User User `gorm:"foreignKey:OwnerID;references:StudentID" json:"user"` // 关联到用户模型，使用学号作为外键
}
