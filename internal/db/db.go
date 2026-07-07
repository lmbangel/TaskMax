package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"taskmax/internal/config"
	"taskmax/internal/models"
)

// NewDB returns a *gorm.DB for the driver named in the config. Unknown driver
// types fall back to a local SQLite database so the app always starts.
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	switch cfg.Database.Type {
	case "sqlite":
		return gorm.Open(sqlite.Open(cfg.Database.DSN), gormCfg)
	case "postgres":
		return gorm.Open(postgres.Open(cfg.Database.DSN), gormCfg)
	case "mysql":
		return gorm.Open(mysql.Open(cfg.Database.DSN), gormCfg)
	default:
		return gorm.Open(sqlite.Open("tasks.db"), gormCfg)
	}
}

// Migrate runs auto-migration for all application models.
func Migrate(gdb *gorm.DB) error {
	return gdb.AutoMigrate(&models.Task{}, &models.PomodoroSession{})
}
