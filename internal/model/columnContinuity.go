package model

type ColumnContinuity struct {
	FkUserColumn
	Continuity
}
type FkUserColumn struct {
	UserID   uint `gorm:"not null;index:idx_user_column,unique" json:"-"`
	ColumnID uint `gorm:"not null;index:idx_user_column,unique" json:"-"`
}
