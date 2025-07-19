package model

type User struct {
	Model
	StudentID string `gorm:"type:varchar(20);uniqueIndex;not null"`
	Password  string `gorm:"type:varchar(255);not null"`
	RoleID    int    `gorm:"default:1;not null"`
	NickName  string `gorm:"type:varchar(20);not null"`
}
