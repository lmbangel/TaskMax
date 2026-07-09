package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"gorm.io/gorm"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"taskmax/internal/config"
	"taskmax/internal/db"
	"taskmax/internal/models"
	"taskmax/internal/services"
)

// Widget window dimensions, shared with the wails.Run options in main.go.
const (
	windowWidth  = 380
	windowHeight = 600
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
	backup   *services.BackupService

	visMu  sync.Mutex
	hidden bool // window hidden to the tray (tracked for the global hotkey)

	lastDueNotify string // local date ("2006-01-02") of the last due-task reminder
}

// NewApp wires up the services around an open database connection.
func NewApp(cfg *config.Config, cfgPath string, gdb *gorm.DB) *App {
	return &App{
		cfg:      cfg,
		cfgPath:  cfgPath,
		db:       gdb,
		tasks:    services.NewTaskService(gdb),
		pomodoro: services.NewPomodoroService(gdb, cfg),
		backup:   services.NewBackupService(gdb),
	}
}

// startup is invoked by Wails once the runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.pomodoro.SetContext(ctx)
	a.restoreWindowPosition()
	a.startTray()
	a.registerHotkey()
	a.startMCP()
	go a.dueReminderLoop()
	go cleanupUpdateArtifacts()
}

// parseDueDate accepts a YYYY-MM-DD date for task due dates.
func parseDueDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid due_date %q — use YYYY-MM-DD", s)
	}
	return t, nil
}

// dueReminderLoop notifies once per day about tasks that are due today or
// overdue. It checks shortly after launch and then every six hours.
func (a *App) dueReminderLoop() {
	time.Sleep(15 * time.Second)
	for {
		a.notifyDueTasks()
		time.Sleep(6 * time.Hour)
	}
}

func (a *App) notifyDueTasks() {
	today := time.Now().Format("2006-01-02")
	if a.lastDueNotify == today {
		return
	}
	now := time.Now()
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	var count int64
	a.db.Model(&models.Task{}).
		Where("status != ? AND due_date IS NOT NULL AND due_date <= ?", "done", endOfToday).
		Count(&count)
	if count == 0 {
		return
	}
	a.lastDueNotify = today
	label := "tasks are"
	if count == 1 {
		label = "task is"
	}
	_ = beeep.Notify("🦆 TaskMax", fmt.Sprintf("%d %s due today or overdue.", count, label), "")
}

// shutdown is invoked by Wails when the application is quitting.
func (a *App) shutdown(ctx context.Context) {
	a.SaveWindowPosition()
	a.stopTray()
}

// restoreWindowPosition puts the widget back where the user last left it,
// or docks it bottom-right on first run / if the stored position is junk.
func (a *App) restoreWindowPosition() {
	w := a.cfg.Window
	if w.Saved && positionSane(w.X, w.Y) {
		wailsruntime.WindowSetPosition(a.ctx, w.X, w.Y)
		return
	}
	a.dockToBottomRight()
}

// SaveWindowPosition persists the current window position. Called by the
// frontend before hiding to the tray and by the shutdown hook, so both exit
// paths remember where the widget was.
func (a *App) SaveWindowPosition() {
	x, y := wailsruntime.WindowGetPosition(a.ctx)
	if !positionSane(x, y) {
		return
	}
	a.cfg.Window = config.WindowConfig{X: x, Y: y, Saved: true}
	_ = config.Save(a.cfgPath, a.cfg)
}

// positionSane rejects the -32000 coordinates Windows reports for minimised
// windows and anything absurdly far outside a plausible virtual desktop.
func positionSane(x, y int) bool {
	return x > -10000 && x < 20000 && y > -10000 && y < 20000
}

// dockToBottomRight parks the widget near the bottom-right corner of the
// primary screen, just above the taskbar.
func (a *App) dockToBottomRight() {
	screens, err := wailsruntime.ScreenGetAll(a.ctx)
	if err != nil || len(screens) == 0 {
		return
	}
	screen := screens[0]
	for _, s := range screens {
		if s.IsPrimary {
			screen = s
			break
		}
	}
	const marginX, marginY = 16, 72 // marginY leaves room for the taskbar
	x := screen.Size.Width - windowWidth - marginX
	y := screen.Size.Height - windowHeight - marginY
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	wailsruntime.WindowSetPosition(a.ctx, x, y)
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

// ----- Comments -----

// GetComments returns a task's comment trail, oldest first.
func (a *App) GetComments(taskID uint) ([]models.Comment, error) {
	return a.tasks.CommentsForTask(taskID)
}

// AddComment appends a user-written comment to a task.
func (a *App) AddComment(taskID uint, body string) (models.Comment, error) {
	return a.tasks.AddComment(taskID, body, "", "")
}

// DeleteComment removes a single comment from a task's trail.
func (a *App) DeleteComment(id uint) error {
	return a.tasks.DeleteComment(id)
}

// ----- Pomodoro -----

// StartPomodoro starts or resumes a countdown for a task. Focusing on a
// "todo" task moves it to "in_progress" so the Doing tab reflects reality.
func (a *App) StartPomodoro(taskID uint, sessionType string) error {
	if (sessionType == "" || sessionType == services.SessionWork) && taskID != 0 {
		_ = a.tasks.SetInProgressIfTodo(taskID)
	}
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

// GetDailyActivity returns per-day completed work sessions for the activity
// heatmap (last `days` days; days without sessions are omitted).
func (a *App) GetDailyActivity(days int) ([]services.DailyActivity, error) {
	return a.pomodoro.DailyActivityRange(days)
}

// ----- Window visibility -----

// HideToTray remembers the window position and hides the widget to the tray.
func (a *App) HideToTray() {
	a.SaveWindowPosition()
	a.visMu.Lock()
	a.hidden = true
	a.visMu.Unlock()
	wailsruntime.WindowHide(a.ctx)
}

// toggleVisibility flips between hidden-in-tray and visible; bound to the
// global hotkey.
func (a *App) toggleVisibility() {
	a.visMu.Lock()
	hidden := a.hidden
	a.visMu.Unlock()
	if hidden {
		a.showWindow()
	} else {
		a.HideToTray()
	}
}

// GetLaunchOnStartup reports whether TaskMax starts with the OS.
func (a *App) GetLaunchOnStartup() bool {
	enabled, err := launchOnStartupEnabled()
	return err == nil && enabled
}

// SetLaunchOnStartup enables or disables starting TaskMax with the OS.
func (a *App) SetLaunchOnStartup(enabled bool) error {
	return setLaunchOnStartup(enabled)
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
