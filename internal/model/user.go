package model

type User struct {
	Model
	StudentID string `gorm:"type:varchar(20);uniqueIndex;not null" json:"student_id"`
	Name      string `gorm:"type:varchar(20);not null" json:"name"`
	RoleID    int    `gorm:"default:0;not null" json:"role_id"`
	NickName  string `gorm:"type:varchar(20);" json:"nick_name"`
	Avatar    string `gorm:"type:varchar(255);" json:"avatar"`
	College   string `gorm:"type:varchar(255);" json:"college"`
	Major     string `gorm:"type:varchar(255);" json:"major"`
	Grade     string `gorm:"type:varchar(10);" json:"grade"`
	// Password 不对外返回，仅用于登录校验与修改密码
	Password string `gorm:"type:varchar(255);" json:"-"`
}
