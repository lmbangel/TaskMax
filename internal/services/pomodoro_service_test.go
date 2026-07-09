package services

import (
	"testing"
	"time"

	"taskmax/internal/config"
	"taskmax/internal/models"
)

func testConfig() *config.Config {
	return &config.Config{
		Pomodoro: config.PomodoroConfig{
			WorkDuration:       25,
			ShortBreak:         5,
			LongBreak:          15,
			SessionsBeforeLong: 4,
		},
	}
}

func TestDurationFor(t *testing.T) {
	s := NewPomodoroService(newTestDB(t), testConfig())

	cases := map[string]int{
		SessionWork:       25,
		SessionShortBreak: 5,
		SessionLongBreak:  15,
		"unknown":         25, // falls back to work duration
	}
	for sessionType, want := range cases {
		if got := s.durationFor(sessionType); got != want {
			t.Errorf("durationFor(%q) = %d, want %d", sessionType, got, want)
		}
	}
}

func TestNextSessionTypeCycle(t *testing.T) {
	s := NewPomodoroService(newTestDB(t), testConfig())

	// Breaks always lead back to work.
	if got := s.nextSessionType(SessionShortBreak); got != SessionWork {
		t.Errorf("after short break: %q, want work", got)
	}
	if got := s.nextSessionType(SessionLongBreak); got != SessionWork {
		t.Errorf("after long break: %q, want work", got)
	}

	// Work sessions 1-3 earn a short break; the 4th a long one.
	for i := 1; i <= 4; i++ {
		s.completedInCycle = i
		want := SessionShortBreak
		if i == 4 {
			want = SessionLongBreak
		}
		if got := s.nextSessionType(SessionWork); got != want {
			t.Errorf("after %d work sessions: %q, want %q", i, got, want)
		}
	}
}

func TestTodayStatsCountsOnlyCompletedToday(t *testing.T) {
	db := newTestDB(t)
	s := NewPomodoroService(db, testConfig())

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: true, StartedAt: now})
	db.Create(&models.PomodoroSession{Type: SessionShortBreak, Duration: 5, Completed: true, StartedAt: now})
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: false, StartedAt: now})
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: true, StartedAt: yesterday})

	stats := s.TodayStats()
	if stats.SessionsCompleted != 2 {
		t.Errorf("SessionsCompleted = %d, want 2", stats.SessionsCompleted)
	}
	if stats.WorkSessions != 1 {
		t.Errorf("WorkSessions = %d, want 1", stats.WorkSessions)
	}
	if stats.TotalFocusMinutes != 25 {
		t.Errorf("TotalFocusMinutes = %d, want 25", stats.TotalFocusMinutes)
	}
}

func TestDailyActivityRange(t *testing.T) {
	db := newTestDB(t)
	s := NewPomodoroService(db, testConfig())

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	longAgo := now.AddDate(0, 0, -200)
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: true, StartedAt: now})
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: true, StartedAt: now})
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: true, StartedAt: twoDaysAgo})
	db.Create(&models.PomodoroSession{Type: SessionShortBreak, Duration: 5, Completed: true, StartedAt: now})
	db.Create(&models.PomodoroSession{Type: SessionWork, Duration: 25, Completed: true, StartedAt: longAgo})

	days, err := s.DailyActivityRange(112)
	if err != nil {
		t.Fatalf("DailyActivityRange: %v", err)
	}
	if len(days) != 2 {
		t.Fatalf("got %d active days, want 2 (breaks and out-of-range excluded)", len(days))
	}

	byDate := map[string]DailyActivity{}
	for _, d := range days {
		byDate[d.Date] = d
	}
	today := now.Local().Format("2006-01-02")
	if d := byDate[today]; d.Count != 2 || d.Minutes != 50 {
		t.Errorf("today = %+v, want Count 2 / Minutes 50", d)
	}
}

func TestStateReturnsSnapshot(t *testing.T) {
	s := NewPomodoroService(newTestDB(t), testConfig())

	state := s.State()
	if state.IsRunning {
		t.Error("fresh service should not be running")
	}
	if state.SessionType != SessionWork {
		t.Errorf("initial session type = %q, want work", state.SessionType)
	}
}
