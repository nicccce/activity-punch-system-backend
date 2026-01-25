package model

// TotalScore actually TotalScoreInCertainActivity 总分数 需注意默认值
// todo: 打分/撤销打分的时候记得更新,你那里很容易获取activity id的啊啊啊
// mysql你最好是A好了
type TotalScore struct {
	FkUserColumn
	Score uint                     `gorm:"not null" json:"score"`
	User  partialUserForTotalScore `gorm:"foreignKey:UserID;references:ID" json:"user"`
}

// 防前端
type partialUserForTotalScore struct {
	ID        uint   `gorm:"primaryKey" json:"user_id"`
	StudentID string `gorm:"type:varchar(20);uniqueIndex;not null" json:"student_id"`
	RoleID    int    `gorm:"default:0;not null" json:"role_id"`
	NickName  string `gorm:"type:varchar(20);not null" json:"nick_name"`
	Avatar    string `gorm:"type:varchar(255);" json:"avatar"`
	College   string `gorm:"type:varchar(255);" json:"college"`
	Major     string `gorm:"type:varchar(255);" json:"major"`
	Grade     string `gorm:"type:varchar(10);" json:"grade"`
}

func (partialUserForTotalScore) TableName() string {
	return "user"
}
