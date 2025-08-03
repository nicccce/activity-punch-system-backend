package model

type User struct {
	Model
	StudentID string `gorm:"type:varchar(20);uniqueIndex;not null" json:"student_id"`
	RoleID    int    `gorm:"default:0;not null" json:"role_id"`
	NickName  string `gorm:"type:varchar(20);not null" json:"nick_name"`
	Avatar    string `gorm:"type:varchar(255);" json:"avatar"`
	Password  string `gorm:"type:varchar(255);" json:"password"`
}
