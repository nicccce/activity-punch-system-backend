package model

type Punch struct {
	Model
	ID        int    `gorm:"primaryKey" json:"id"`
	ColumnID  int    `gorm:"not null" json:"column_id"`
	UserID    string `gorm:"not null" json:"user_id"`
	Content   string `gorm:"type:varchar(255);not null" json:"content"`
	Status    int    `gorm:"not null" json:"status"` //status为0 待审核 1 审核通过 2 不通过 3 已删除
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"updated_at"`
}
