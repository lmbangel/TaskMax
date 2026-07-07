package main

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"taskmax/internal/config"
	"taskmax/internal/db"
	"taskmax/internal/models"
	"taskmax/internal/services"
)

// App is the Wails-bound application struct. Every exported method here is
// callable from the Svelte frontend via window.go.main.App.MethodName().
type App struct {
	ctx      context.Context
	cfg      *config.Config
	cfgPath  string
	db       *gorm.DB
	tasks    *services.TaskService
	pomodoro *services.PomodoroService
}

// NewApp wires up the services around an open database connection.
func NewApp(cfg *config.Config, cfgPath string, gdb *gorm.DB) *App {
	return &App{
		cfg:      cfg,
		cfgPath:  cfgPath,
		db:       gdb,
		tasks:    services.NewTaskService(gdb),
		pomodoro: services.NewPomodoroService(gdb, cfg),
	}
}

// startup is invoked by Wails once the runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.pomodoro.SetContext(ctx)
}

// ----- Tasks -----

// GetAllTasks returns every task in display order.
func (a *App) GetAllTasks() ([]models.Task, error) {
	return a.tasks.GetAll()
}

// GetTasksByStatus returns tasks filtered by status.
func (a *App) GetTasksByStatus(status string) ([]models.Task, error) {
	return a.tasks.GetByStatus(status)
}

// CreateTask inserts a new task.
func (a *App) CreateTask(task models.Task) (models.Task, error) {
	return a.tasks.Create(task)
}

// UpdateTask saves changes to an existing task.
func (a *App) UpdateTask(task models.Task) (models.Task, error) {
	return a.tasks.Update(task)
}

// DeleteTask removes a task and its sessions.
func (a *App) DeleteTask(id uint) error {
	return a.tasks.Delete(id)
}

// ReorderTasks persists a new task ordering (list of IDs in display order).
func (a *App) ReorderTasks(orderedIDs []uint) error {
	return a.tasks.Reorder(orderedIDs)
}

// ----- Pomodoro -----

// StartPomodoro starts or resumes a countdown for a task.
func (a *App) StartPomodoro(taskID uint, sessionType string) error {
	return a.pomodoro.Start(taskID, sessionType)
}

// StopPomodoro pauses the active countdown.
func (a *App) StopPomodoro() error {
	return a.pomodoro.Stop()
}

// GetTimerState returns the current timer snapshot.
func (a *App) GetTimerState() services.TimerState {
	return a.pomodoro.State()
}

// GetSessionsForTask returns the pomodoro history for a task.
func (a *App) GetSessionsForTask(taskID uint) ([]models.PomodoroSession, error) {
	return a.pomodoro.SessionsForTask(taskID)
}

// GetTodayStats returns today's focus statistics.
func (a *App) GetTodayStats() services.PomodoroStats {
	return a.pomodoro.TodayStats()
}

// ----- Config -----

// GetConfig returns the current application configuration.
func (a *App) GetConfig() config.Config {
	return *a.cfg
}

// SaveConfig persists the given configuration to disk and updates the running
// copy. Timing changes take effect on the next session; a database-driver
// change requires an app restart (reported back to the user by the frontend).
func (a *App) SaveConfig(cfg config.Config) error {
	if err := config.Save(a.cfgPath, &cfg); err != nil {
		return err
	}
	*a.cfg = cfg
	return nil
}

// TestConnection attempts to open (and ping) a database with the given driver
// and DSN, without disturbing the live connection. Used by the Settings panel.
func (a *App) TestConnection(dbType, dsn string) error {
	testCfg := &config.Config{
		Database: config.DatabaseConfig{Type: dbType, DSN: dsn},
	}
	conn, err := db.NewDB(testCfg)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		return fmt.Errorf("could not access database handle: %w", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("could not reach database: %w", err)
	}
	return nil
}
