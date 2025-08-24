package model

import (
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint           `gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
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
