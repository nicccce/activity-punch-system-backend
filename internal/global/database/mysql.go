package database

import (
	"activity-punch-system-backend/config"
	"activity-punch-system-backend/internal/global/otel"
	"activity-punch-system-backend/internal/model"
	"activity-punch-system-backend/tools"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

func Init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Get().Mysql.Username,
		config.Get().Mysql.Password,
		config.Get().Mysql.Host,
		config.Get().Mysql.Port,
		config.Get().Mysql.DBName,
	)

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, // 使用单数表名
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
	tools.PanicOnErr(DB.Use(otel.GetGormPlugin()))
	tools.PanicOnErr(DB.AutoMigrate(model.User{}))
}
