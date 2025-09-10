package model

import (
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint           `gorm:"primaryKey" excel:"id"`
	CreatedAt time.Time      `json:"created_at"  excel:"创建时间"`
	UpdatedAt time.Time      `json:"updated_at" excel:"更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at" excel:"-"`
}

func (m *Model) CreateTime() int64 {
	return m.CreatedAt.UnixMilli()
}

func (m *Model) UpdateTime() int64 {
	return m.UpdatedAt.UnixMilli()
}

type Dto struct {
	ID         uint  `json:"id"`
	CreateTime int64 `json:"create_time"`
	UpdateTime int64 `json:"update_time"`
}
