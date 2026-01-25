package database

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/model"
	"activity-punch-system/tools"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// autoMigrateModels 定义需要自动迁移的模型列表
var autoMigrateModels = []any{
	&model.User{},
	&model.Activity{},
	&model.Project{},
	&model.Column{},
	&model.Punch{},
	&model.PunchImg{},
	&model.Star{},
	&model.TotalScore{},
	&model.Score{},
	&model.ActivityContinuity{},
	&model.ColumnContinuity{},
	&model.ProjectContinuity{},
	&model.IntegrityRule{},
	&model.ContinuityRule{},
	// 在这里添加其他模型
}

func Init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Get().Mysql.Username,
		config.Get().Mysql.Password,
		config.Get().Mysql.Host,
		config.Get().Mysql.Port,
		config.Get().Mysql.DBName,
	)
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, // 还是单数表名好
	}

	switch config.Get().Mode {
	case config.ModeDebug:
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	case config.ModeRelease:
		gormConfig.Logger = logger.Discard
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	tools.PanicOnErr(err)
	DB = db

	// 使用模型列表进行自动迁移
	tools.PanicOnErr(DB.AutoMigrate(autoMigrateModels...))
}
