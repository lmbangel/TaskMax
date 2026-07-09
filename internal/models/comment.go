package models

import (
	"gorm.io/gorm"
)

// Comment is a timestamped note on a task. Comments build a trail the
// description can't: who did what, when, and why — especially useful when
// coding agents work a task over MCP alongside the user.
type Comment struct {
	gorm.Model
	TaskID uint   `json:"task_id"`
	Body   string `json:"body"`
	Source string `json:"source"` // "" = written in the UI, "agent" = via MCP
	Author string `json:"author"` // optional display name, e.g. "Claude Code"
}
