package services

import (
	"context"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"

	"taskmax/internal/config"
	"taskmax/internal/models"
)

// Session type constants.
const (
	SessionWork       = "work"
	SessionShortBreak = "short_break"
	SessionLongBreak  = "long_break"
)

// Wails runtime event names emitted to the frontend.
const (
	EventTick     = "pomodoro:tick"
	EventComplete = "pomodoro:complete"
	EventStopped  = "pomodoro:stopped"
)

// TimerState is the snapshot the frontend polls every second.
type TimerState struct {
	SecondsRemaining int    `json:"seconds_remaining"`
	SessionType      string `json:"session_type"`
	IsRunning        bool   `json:"is_running"`
	ActiveTaskID     uint   `json:"active_task_id"`
}

// PomodoroStats summarises a day's focus activity.
type PomodoroStats struct {
	SessionsCompleted int `json:"sessions_completed"`
	WorkSessions      int `json:"work_sessions"`
	TotalFocusMinutes int `json:"total_focus_minutes"`
}

// CompletePayload accompanies the "pomodoro:complete" event so the frontend
// knows what to advance to.
type CompletePayload struct {
	FinishedType string `json:"finished_type"`
	NextType     string `json:"next_type"`
	TaskID       uint   `json:"task_id"`
	NextDuration int    `json:"next_duration"` // minutes
}

// PomodoroService owns the countdown goroutine and session persistence.
type PomodoroService struct {
	db  *gorm.DB
	cfg *config.Config
	ctx context.Context

	mu               sync.Mutex
	state            TimerState
	cancel           context.CancelFunc
	activeSessionID  uint
	completedInCycle int // completed work sessions since the last long break
}

// NewPomodoroService constructs a service. Call SetContext once the Wails
// runtime context is available (during OnStartup) so events can be emitted.
func NewPomodoroService(db *gorm.DB, cfg *config.Config) *PomodoroService {
	return &PomodoroService{
		db:  db,
		cfg: cfg,
		state: TimerState{
			SessionType: SessionWork,
		},
	}
}

// SetContext stores the Wails runtime context used for event emission.
func (s *PomodoroService) SetContext(ctx context.Context) {
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()
}

// durationFor returns the configured length (in minutes) for a session type.
func (s *PomodoroService) durationFor(sessionType string) int {
	switch sessionType {
	case SessionShortBreak:
		return s.cfg.Pomodoro.ShortBreak
	case SessionLongBreak:
		return s.cfg.Pomodoro.LongBreak
	default:
		return s.cfg.Pomodoro.WorkDuration
	}
}

// Start begins (or resumes) a countdown for the given task and session type.
// Any session already running is cancelled first.
func (s *PomodoroService) Start(taskID uint, sessionType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionType == "" {
		sessionType = SessionWork
	}

	// Cancel any existing goroutine before starting a new one.
	s.stopLocked(false)

	// Resume support: if the same task/type was paused mid-session, continue
	// from where it left off instead of resetting.
	resuming := !s.state.IsRunning &&
		s.state.SecondsRemaining > 0 &&
		s.state.SessionType == sessionType &&
		s.state.ActiveTaskID == taskID &&
		s.activeSessionID != 0

	if !resuming {
		duration := s.durationFor(sessionType)
		session := models.PomodoroSession{
			TaskID:    taskID,
			Type:      sessionType,
			Duration:  duration,
			Completed: false,
			StartedAt: time.Now(),
		}
		if err := s.db.Create(&session).Error; err != nil {
			return err
		}
		s.activeSessionID = session.ID
		s.state = TimerState{
			SecondsRemaining: duration * 60,
			SessionType:      sessionType,
			IsRunning:        true,
			ActiveTaskID:     taskID,
		}
	} else {
		s.state.IsRunning = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.run(ctx)
	return nil
}

// Stop pauses the current countdown, keeping the remaining time so it can be
// resumed. The goroutine is cancelled cleanly via context.
func (s *PomodoroService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked(true)
	return nil
}

// stopLocked cancels the running goroutine. Caller must hold s.mu.
func (s *PomodoroService) stopLocked(emit bool) {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.state.IsRunning {
		s.state.IsRunning = false
		if emit {
			s.emit(EventStopped, s.state)
		}
	}
}

// run is the countdown loop driven by a one-second ticker.
func (s *PomodoroService) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.state.IsRunning {
				s.mu.Unlock()
				return
			}
			s.state.SecondsRemaining--
			remaining := s.state.SecondsRemaining
			snapshot := s.state
			s.mu.Unlock()

			s.emit(EventTick, snapshot)

			if remaining <= 0 {
				s.complete(ctx)
				return
			}
		}
	}
}

// complete finalises the current session, notifies the user, and emits the
// "pomodoro:complete" event with the next session to advance to.
func (s *PomodoroService) complete(ctx context.Context) {
	s.mu.Lock()

	finishedType := s.state.SessionType
	taskID := s.state.ActiveTaskID
	sessionID := s.activeSessionID

	// Persist completion.
	now := time.Now()
	if sessionID != 0 {
		s.db.Model(&models.PomodoroSession{}).
			Where("id = ?", sessionID).
			Updates(map[string]interface{}{
				"completed":    true,
				"completed_at": &now,
			})
	}

	// Work sessions count toward the task and the long-break cycle.
	if finishedType == SessionWork {
		if taskID != 0 {
			s.db.Model(&models.Task{}).
				Where("id = ?", taskID).
				UpdateColumn("pomodoro_count", gorm.Expr("pomodoro_count + 1"))
		}
		s.completedInCycle++
	}

	nextType := s.nextSessionType(finishedType)
	nextDuration := s.durationFor(nextType)

	s.state = TimerState{
		SecondsRemaining: 0,
		SessionType:      nextType,
		IsRunning:        false,
		ActiveTaskID:     taskID,
	}
	s.activeSessionID = 0
	s.cancel = nil
	s.mu.Unlock()

	// Desktop notification and chime (best-effort — failures are non-fatal).
	title, body := notificationText(finishedType, nextType)
	_ = beeep.Notify(title, body, "")
	if s.cfg.Pomodoro.Sound {
		go func() {
			_ = beeep.Beep(660, 250)
			_ = beeep.Beep(880, 350)
		}()
	}

	s.emit(EventComplete, CompletePayload{
		FinishedType: finishedType,
		NextType:     nextType,
		TaskID:       taskID,
		NextDuration: nextDuration,
	})
}

// nextSessionType implements the work → short break → … → long break cycle.
func (s *PomodoroService) nextSessionType(finished string) string {
	if finished == SessionWork {
		if s.completedInCycle > 0 && s.completedInCycle%s.cfg.Pomodoro.SessionsBeforeLong == 0 {
			return SessionLongBreak
		}
		return SessionShortBreak
	}
	// After any break, return to work.
	return SessionWork
}

// State returns a copy of the current timer state.
func (s *PomodoroService) State() TimerState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// SessionsForTask returns all sessions logged against a task, newest first.
func (s *PomodoroService) SessionsForTask(taskID uint) ([]models.PomodoroSession, error) {
	var sessions []models.PomodoroSession
	err := s.db.Where("task_id = ?", taskID).
		Order("started_at desc").
		Find(&sessions).Error
	return sessions, err
}

// TodayStats aggregates completed sessions since local midnight.
func (s *PomodoroService) TodayStats() PomodoroStats {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var sessions []models.PomodoroSession
	s.db.Where("completed = ? AND started_at >= ?", true, start).Find(&sessions)

	stats := PomodoroStats{}
	for _, sess := range sessions {
		stats.SessionsCompleted++
		if sess.Type == SessionWork {
			stats.WorkSessions++
			stats.TotalFocusMinutes += sess.Duration
		}
	}
	return stats
}

// DailyActivity is one day's completed focus work, for the activity heatmap.
type DailyActivity struct {
	Date    string `json:"date"` // local date, "2006-01-02"
	Count   int    `json:"count"`
	Minutes int    `json:"minutes"`
}

// DailyActivityRange aggregates completed work sessions per local day for the
// last `days` days (today included). Days without activity are omitted.
func (s *PomodoroService) DailyActivityRange(days int) ([]DailyActivity, error) {
	if days <= 0 || days > 400 {
		days = 112
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -(days - 1))

	var sessions []models.PomodoroSession
	err := s.db.Where("completed = ? AND type = ? AND started_at >= ?", true, SessionWork, start).
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}

	byDay := map[string]*DailyActivity{}
	for _, sess := range sessions {
		key := sess.StartedAt.Local().Format("2006-01-02")
		d, ok := byDay[key]
		if !ok {
			d = &DailyActivity{Date: key}
			byDay[key] = d
		}
		d.Count++
		d.Minutes += sess.Duration
	}

	out := make([]DailyActivity, 0, len(byDay))
	for _, d := range byDay {
		out = append(out, *d)
	}
	return out, nil
}

// emit sends a Wails runtime event if the context has been wired up.
func (s *PomodoroService) emit(event string, data interface{}) {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, event, data)
}

// notificationText builds a friendly desktop notification for a transition.
func notificationText(finished, next string) (string, string) {
	switch finished {
	case SessionWork:
		if next == SessionLongBreak {
			return "🦆 Work session complete!", "Great focus — time for a long break 🌿"
		}
		return "🦆 Work session complete!", "Nicely done — take a short break ☕"
	case SessionShortBreak, SessionLongBreak:
		return "Break's over ✨", "Ready to focus? Starting your next work session 🦆"
	default:
		return "TaskMax", "Session complete."
	}
}
