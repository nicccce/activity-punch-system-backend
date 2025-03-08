package model

type User struct {
	Model
	StudentID string `gorm:"type:varchar(20);uniqueIndex;not null"`
	RoleID    int    `gorm:"default:1;not null"`
	RealName  string `gorm:"type:varchar(20);not null"`
}
