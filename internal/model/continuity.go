package model

import "time"

// Continuity actually in certain activity 打卡连续天数等 需注意默认值
// todo: 打卡的时候记得更新
type Continuity struct {
	UserID     uint                     `gorm:"not null;index:idx_user_activity,unique" json:"-"`
	ActivityID uint                     `gorm:"not null;index:idx_user_activity,unique" json:"activity_id"`
	Current    uint                     `gorm:"not null" json:"current"`
	Max        uint                     `gorm:"not null" json:"max"`
	User       partialUserForTotalScore `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Total      uint                     `gorm:"not null" json:"total"`
	EndAt      int64                    `gorm:"not null" json:"-"`
	//RefreshAt time.Time `gorm:"not null" json:"refresh_at"`
}

// AddTo 仅仅是更新连续天数
func (c *Continuity) AddTo(toTime time.Time) {
	day := toTime.Unix() / (24 * 60 * 6)
	if c.EndAt >= day {
		return
	}
	c.Current += uint(1 + day - c.EndAt)
	if c.Current > c.Max {
		c.Max = c.Current
	}
	c.EndAt = day
}
