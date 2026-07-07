package models

import (
	"time"

	"gorm.io/gorm"
)

// Task represents a single unit of work the user wants to track.
type Task struct {
	gorm.Model
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Priority      string     `json:"priority"` // "low", "medium", "high"
	Status        string     `json:"status"`   // "todo", "in_progress", "done"
	Tags          string     `json:"tags"`     // comma-separated
	DueDate       *time.Time `json:"due_date"`
	PomodoroCount int        `json:"pomodoro_count"` // how many work sessions logged against it
	Position      int        `json:"position"`       // manual sort order for drag-to-reorder
}
