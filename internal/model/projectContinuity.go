package model

type ProjectContinuity struct {
	FkUserProject
	Continuity
}
type FkUserProject struct {
	UserID    uint `gorm:"not null;index:idx_user_project,unique" json:"-"`
	ProjectID uint `gorm:"not null;index:idx_user_project,unique" json:"-"`
}
