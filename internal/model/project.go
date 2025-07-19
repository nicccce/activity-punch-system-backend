package model

type Project struct {
	Model
	Name        string `gorm:"type:varchar(100);not null"` // 项目名称
	Description string `gorm:"type:varchar(255);"`         // 项目描述
	OwnerID     int    `gorm:"not null"`                   // 所有者ID，
	StartDate   int64  `gorm:""`                           // 项目开始时间
	EndDate     int64  `gorm:""`                           // 项目结束时间
	Category    string `gorm:"type:varchar(50);not null"`  // 项目类别（如暑期打卡、寒假打卡等）
	// 关联到用户
	User      User  `gorm:"foreignKey:OwnerID;references:ID"` // 关联到用户模型
	CreatedAt int64 `gorm:"autoCreateTime"`                   // 创建时间
	UpdatedAt int64 `gorm:"autoUpdateTime"`                   // 更新时间
	DeletedAt int64 `gorm:"index"`                            // 删除时间，软删除
}
