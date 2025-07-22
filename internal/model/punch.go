package model

type Punch struct {
	Model
	ID       int    `gorm:"primaryKey" json:"id"`
	ColumnID int    `gorm:"not null" json:"column_id"`
	UserID   string `gorm:"not null" json:"user_id"`
	Content  string `gorm:"type:varchar(255);not null" json:"content"`
	Status   int    `gorm:"not null" json:"status"` //status为  0 待审核   1 审核通过   2 不通过   3 已删除
}
