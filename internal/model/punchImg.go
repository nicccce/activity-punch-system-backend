package model

type PunchImg struct {
	Model
	ID       int    `gorm:"primaryKey;autoIncrement" json:"id"`        // 自增ID
	ColumnID int    `gorm:"not null" json:"column_id"`                 // 关联的栏目ID
	ImgURL   string `gorm:"type:varchar(255);not null" json:"img_url"` // 图片URL
	PunchID  int    `gorm:"not null" json:"punch_id"`                  // 关联的打卡ID

	// 关联到用户
	Punch Punch `gorm:"foreignKey:PunchID;references:ID" json:"punch"` // 关联到用户模型，使用学号作为外键
}
