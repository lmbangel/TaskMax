package main

import (
	"strconv"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// The taskmax:// URL scheme carries "which task" through a notification
// click: Windows launches the protocol handler, the single-instance guard
// forwards the URL to the running app, and the widget navigates to the task.
const taskURLPrefix = "taskmax://task/"

// notifyAgentActivity shows a desktop notification for something an agent
// did over MCP. Clicking it (on Windows) brings the widget up on that task.
// Muted by the "Notify when agents create or complete tasks" setting.
func (a *App) notifyAgentActivity(title, body string, taskID uint) {
	if !a.cfg.App.AgentNotifications {
		return
	}
	pushAgentToast(title, body, taskID)
}

// notifyTaskCompleted announces an agent finishing a task. The latest
// comment — usually the agent's summary of the work — becomes the preview
// line, so the toast itself tells you what was done.
func (a *App) notifyTaskCompleted(title string, taskID uint) {
	body := "Click to see the work."
	if comments, err := a.tasks.CommentsForTask(taskID); err == nil && len(comments) > 0 {
		body = comments[len(comments)-1].Body
		if len(body) > 140 {
			body = body[:137] + "..."
		}
	}
	a.notifyAgentActivity("🤖 Task completed: "+title, body, taskID)
}

// taskIDFromArgs finds a taskmax://task/<id> URL among process arguments.
// Returns 0 when there is none (a normal launch).
func taskIDFromArgs(args []string) uint {
	for _, arg := range args {
		if !strings.HasPrefix(arg, taskURLPrefix) {
			continue
		}
		rest := strings.Trim(strings.TrimPrefix(arg, taskURLPrefix), "/")
		id, err := strconv.ParseUint(rest, 10, 32)
		if err != nil {
			return 0
		}
		return uint(id)
	}
	return 0
}

// navigateToTask brings the widget into view on the given task's detail.
func (a *App) navigateToTask(id uint) {
	a.showWindow()
	wailsruntime.EventsEmit(a.ctx, "task:navigate", id)
}

// GetPendingNavigation returns the task the app was asked to open at launch
// (cold start from a notification click), or 0. The frontend calls this once
// on mount; the pending state clears so it fires only once.
func (a *App) GetPendingNavigation() uint {
	id := a.pendingNav
	a.pendingNav = 0
	return id
}
