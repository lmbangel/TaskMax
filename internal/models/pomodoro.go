package models

import (
	"time"

	"gorm.io/gorm"
)

// PomodoroSession records a single work or break interval.
type PomodoroSession struct {
	gorm.Model
	TaskID      uint       `json:"task_id"`
	Type        string     `json:"type"`     // "work", "short_break", "long_break"
	Duration    int        `json:"duration"` // minutes
	Completed   bool       `json:"completed"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}
