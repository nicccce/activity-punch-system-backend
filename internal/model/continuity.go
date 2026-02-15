package model

import "time"

// Continuity actually in certain activity 打卡连续天数等 需注意默认值
// todo: 打卡的时候记得更新
type Continuity struct {
	FkUserActivity
	Current uint                     `gorm:"not null" json:"current"`
	Max     uint                     `gorm:"not null" json:"max"`
	User    partialUserForTotalScore `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Total   uint                     `gorm:"not null" json:"total"`
	EndAt   int64                    `gorm:"not null" json:"-"`
	//RefreshAt time.Time `gorm:"not null" json:"refresh_at"`
}
type FkUserActivity struct {
	UserID     uint `gorm:"not null;index:idx_user_activity,unique" json:"-"`
	ActivityID uint `gorm:"not null;index:idx_user_activity,unique;index:idx_activity_score,priority:1" json:"-"`
}

// RefreshTo 仅仅是更新连续天数
func (c *Continuity) RefreshTo(toTime time.Time) {
	// 使用北京时区计算"天"
	loc := time.FixedZone("CST", 8*60*60)
	t := toTime.In(loc)
	// 计算北京时间零点的 Unix 时间戳对应的天数
	dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	day := dayStart.Unix() / (24 * 60 * 60)

	if day-c.EndAt >= 1 {
		c.Total++
		if day-c.EndAt == 1 {
			c.Current++
		} else {
			c.Current = 1
		}
	}
	if c.Current > c.Max {
		c.Max = c.Current
	}
	c.EndAt = day
}
