package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"taskmax/internal/models"
)

// newTestDB opens a fresh in-memory SQLite database with the app's schema.
// MaxOpenConns(1) pins the pool to a single connection — every :memory:
// connection is its own empty database, so a second one would lose the data.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.Task{}, &models.PomodoroSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func mustCreateTask(t *testing.T, s *TaskService, task models.Task) models.Task {
	t.Helper()
	created, err := s.Create(task)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return created
}
