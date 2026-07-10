package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"taskmax/internal/models"
)

// EventTasksChanged tells the frontend an agent mutated tasks via MCP so the
// Svelte store can refresh.
const EventTasksChanged = "tasks:changed"

// startMCP serves an embedded MCP server over streamable HTTP on
// 127.0.0.1:<port>/mcp. Coding agents (Claude Code CLI/VS Code/desktop, or
// any MCP client) manage tasks through the same services the UI uses — no
// screen control, no second database writer, and multiple agent sessions can
// connect concurrently.
func (a *App) startMCP() {
	if !a.cfg.MCP.Enabled {
		return
	}
	s := server.NewMCPServer("TaskMax", version)
	a.registerMCPTools(s)

	httpServer := server.NewStreamableHTTPServer(s)
	addr := fmt.Sprintf("127.0.0.1:%d", a.cfg.MCP.Port)
	go func() {
		if err := httpServer.Start(addr); err != nil {
			log.Printf("mcp server stopped: %v", err)
		}
	}()
	log.Printf("mcp server listening on http://%s/mcp", addr)
}

// tasksChanged notifies the UI after an agent-driven mutation.
func (a *App) tasksChanged() {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, EventTasksChanged)
	}
}

// commentsChanged tells the UI an agent commented on a task, so an open
// detail view can refresh its trail live.
func (a *App) commentsChanged(taskID uint) {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "comments:changed", taskID)
	}
}

func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// taskByID loads a task or returns a tool error result.
func (a *App) taskByID(id int) (models.Task, *mcp.CallToolResult) {
	var task models.Task
	if err := a.db.First(&task, id).Error; err != nil {
		return task, mcp.NewToolResultError(fmt.Sprintf("task %d not found", id))
	}
	return task, nil
}

func (a *App) registerMCPTools(s *server.MCPServer) {
	// ----- create_task -----
	s.AddTool(mcp.NewTool("create_task",
		mcp.WithDescription("Create a new task on the TaskMax board. Returns the created task including its ID."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Short task title")),
		mcp.WithString("description", mcp.Description("Longer free-text description")),
		mcp.WithString("priority", mcp.Description("low | medium | high (default medium)")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags, e.g. \"bug,frontend\"")),
		mcp.WithString("due_date", mcp.Description("Due date as YYYY-MM-DD")),
		mcp.WithString("recurrence", mcp.Description("Repeat rule: daily | weekly | monthly (empty = never)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, err := req.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		task := models.Task{
			Title:       title,
			Description: req.GetString("description", ""),
			Priority:    req.GetString("priority", "medium"),
			Tags:        req.GetString("tags", ""),
			Recurrence:  req.GetString("recurrence", ""),
			Source:      "agent",
		}
		if due := req.GetString("due_date", ""); due != "" {
			t, perr := parseDueDate(due)
			if perr != nil {
				return mcp.NewToolResultError(perr.Error()), nil
			}
			task.DueDate = &t
		}
		created, err := a.tasks.Create(task)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		a.tasksChanged()
		a.notifyAgentActivity("🤖 New task from your agent", created.Title, created.ID)
		return jsonResult(created)
	})

	// ----- list_tasks -----
	s.AddTool(mcp.NewTool("list_tasks",
		mcp.WithDescription("List tasks on the board, optionally filtered by status, tag, or a text search."),
		mcp.WithString("status", mcp.Description("todo | in_progress | done (omit for all)")),
		mcp.WithString("tag", mcp.Description("Only tasks carrying this tag")),
		mcp.WithString("search", mcp.Description("Case-insensitive text match on title/description/tags")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		all, err := a.tasks.GetAll()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		status := req.GetString("status", "")
		tag := strings.ToLower(req.GetString("tag", ""))
		search := strings.ToLower(req.GetString("search", ""))
		out := make([]models.Task, 0, len(all))
		for _, t := range all {
			if status != "" && t.Status != status {
				continue
			}
			if tag != "" && !hasTag(t.Tags, tag) {
				continue
			}
			if search != "" &&
				!strings.Contains(strings.ToLower(t.Title+" "+t.Description+" "+t.Tags), search) {
				continue
			}
			out = append(out, t)
		}
		return jsonResult(out)
	})

	// ----- update_task -----
	s.AddTool(mcp.NewTool("update_task",
		mcp.WithDescription("Update fields on an existing task. Only the provided fields change."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Task ID")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithString("priority", mcp.Description("low | medium | high")),
		mcp.WithString("status", mcp.Description("todo | in_progress | done")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags (replaces existing)")),
		mcp.WithString("due_date", mcp.Description("Due date as YYYY-MM-DD (empty string clears it)")),
		mcp.WithString("recurrence", mcp.Description("daily | weekly | monthly (empty string clears it)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		task, errRes := a.taskByID(id)
		if errRes != nil {
			return errRes, nil
		}
		prevStatus := task.Status
		args := req.GetArguments()
		if v, ok := args["title"].(string); ok && v != "" {
			task.Title = v
		}
		if v, ok := args["description"].(string); ok {
			task.Description = v
		}
		if v, ok := args["priority"].(string); ok && v != "" {
			task.Priority = v
		}
		if v, ok := args["status"].(string); ok && v != "" {
			task.Status = v
		}
		if v, ok := args["tags"].(string); ok {
			task.Tags = v
		}
		if v, ok := args["recurrence"].(string); ok {
			task.Recurrence = v
		}
		if v, ok := args["due_date"].(string); ok {
			if v == "" {
				task.DueDate = nil
			} else {
				t, perr := parseDueDate(v)
				if perr != nil {
					return mcp.NewToolResultError(perr.Error()), nil
				}
				task.DueDate = &t
			}
		}
		nowDone := task.Status == "done" && prevStatus != "done"
		updated, err := a.tasks.Update(task)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		a.tasksChanged()
		if nowDone {
			a.notifyTaskCompleted(updated.Title, updated.ID)
		}
		return jsonResult(updated)
	})

	// ----- complete_task -----
	s.AddTool(mcp.NewTool("complete_task",
		mcp.WithDescription("Mark a task as done. Recurring tasks spring back to todo with the next due date."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Task ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		task, errRes := a.taskByID(id)
		if errRes != nil {
			return errRes, nil
		}
		alreadyDone := task.Status == "done"
		task.Status = "done"
		updated, err := a.tasks.Update(task)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		a.tasksChanged()
		if !alreadyDone {
			a.notifyTaskCompleted(task.Title, task.ID)
		}
		return jsonResult(updated)
	})

	// ----- delete_task -----
	s.AddTool(mcp.NewTool("delete_task",
		mcp.WithDescription("Delete a task and its session history. Irreversible."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Task ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := a.tasks.Delete(uint(id)); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		a.tasksChanged()
		return mcp.NewToolResultText(fmt.Sprintf("task %d deleted", id)), nil
	})

	// ----- add_comment -----
	s.AddTool(mcp.NewTool("add_comment",
		mcp.WithDescription("Add a comment to a task's trail. Use this to leave context and traceability as you work: what was done, decisions made, links to PRs/commits. Comments are timestamped and marked as agent-written."),
		mcp.WithNumber("task_id", mcp.Required(), mcp.Description("Task ID")),
		mcp.WithString("body", mcp.Required(), mcp.Description("The comment text")),
		mcp.WithString("author", mcp.Description("Display name shown on the comment, e.g. \"Claude Code\"")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := req.RequireInt("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := req.RequireString("body")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		comment, err := a.tasks.AddComment(uint(taskID), body, "agent", req.GetString("author", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		a.commentsChanged(uint(taskID))
		return jsonResult(comment)
	})

	// ----- get_comments -----
	s.AddTool(mcp.NewTool("get_comments",
		mcp.WithDescription("Read a task's comment trail, oldest first. Check this for context left by the user or other agent sessions before working on a task."),
		mcp.WithNumber("task_id", mcp.Required(), mcp.Description("Task ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := req.RequireInt("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if _, errRes := a.taskByID(taskID); errRes != nil {
			return errRes, nil
		}
		comments, err := a.tasks.CommentsForTask(uint(taskID))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(comments)
	})

	// ----- start_pomodoro -----
	s.AddTool(mcp.NewTool("start_pomodoro",
		mcp.WithDescription("Start (or resume) a focus session. Focusing on a todo task moves it to in_progress."),
		mcp.WithNumber("task_id", mcp.Description("Task to focus on (0 or omitted = no task)")),
		mcp.WithString("session_type", mcp.Description("work | short_break | long_break (default work)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID := req.GetInt("task_id", 0)
		if err := a.StartPomodoro(uint(taskID), req.GetString("session_type", "work")); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		a.tasksChanged()
		return jsonResult(a.pomodoro.State())
	})

	// ----- stop_pomodoro -----
	s.AddTool(mcp.NewTool("stop_pomodoro",
		mcp.WithDescription("Pause the running focus session (it can be resumed with start_pomodoro)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := a.pomodoro.Stop(); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(a.pomodoro.State())
	})

	// ----- get_timer_state -----
	s.AddTool(mcp.NewTool("get_timer_state",
		mcp.WithDescription("Get the current pomodoro timer state (remaining seconds, session type, active task)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(a.pomodoro.State())
	})

	// ----- get_today_stats -----
	s.AddTool(mcp.NewTool("get_today_stats",
		mcp.WithDescription("Get today's focus statistics: completed sessions, work sessions, total focus minutes."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(a.pomodoro.TodayStats())
	})

	// ----- get_activity -----
	s.AddTool(mcp.NewTool("get_activity",
		mcp.WithDescription("Per-day completed work sessions for the recent past (the data behind the activity heatmap)."),
		mcp.WithNumber("days", mcp.Description("How many days back to include (default 112)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activity, err := a.pomodoro.DailyActivityRange(req.GetInt("days", 112))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(activity)
	})
}

func hasTag(tags, want string) bool {
	for _, t := range strings.Split(tags, ",") {
		if strings.ToLower(strings.TrimSpace(t)) == want {
			return true
		}
	}
	return false
}
