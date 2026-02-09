package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Punch struct {
	Model
	//ID       int    `gorm:"primaryKey" json:"id"`
	ColumnID int    `gorm:"not null" json:"column_id"  excel:"-"`
	UserID   uint   `gorm:"not null" json:"user_id" excel:"-"`
	Content  string `gorm:"type:text;not null" json:"content" excel:"打卡内容"`
	Status   int    `gorm:"not null" json:"status" excel:"审核状态"` //status为  0 待审核   1 审核通过   2 不通过
}

// todo: 打卡能被删除吗？
func (p *Punch) AfterCreate(tx *gorm.DB) (err error) {
	fkUserActivity := tx.Statement.Context.Value("fk_user_activity")
	if fkUserActivity == nil {
		return nil // 如果没有传递 fk_user_activity，跳过连续性更新
	}

	c := Continuity{FkUserActivity: *(fkUserActivity.(*FkUserActivity))}

	// 使用 FOR UPDATE 锁定特定行，避免并发冲突
	err = tx.Model(&Continuity{}).
		Where("activity_id = ? AND user_id = ?", c.ActivityID, c.UserID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&c).Error

	if err != nil {
		return err
	}

	flag := c.Total
	c.RefreshTo(p.CreatedAt)

	if flag == 0 {
		// 首次创建记录
		if createErr := tx.Create(&c).Error; createErr != nil {
			// 如果创建失败（可能是并发导致的重复），尝试更新
			return tx.Model(&Continuity{}).
				Where("activity_id = ? AND user_id = ?", c.ActivityID, c.UserID).
				Updates(map[string]interface{}{
					"current": c.Current,
					"max":     c.Max,
					"total":   c.Total,
					"end_at":  c.EndAt,
				}).Error
		}
		return nil
	}

	// 更新现有记录
	return tx.Model(&Continuity{}).
		Where("activity_id = ? AND user_id = ?", c.ActivityID, c.UserID).
		Updates(map[string]interface{}{
			"current": c.Current,
			"max":     c.Max,
			"total":   c.Total,
			"end_at":  c.EndAt,
		}).Error
}
