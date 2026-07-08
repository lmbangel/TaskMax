package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"taskmax/internal/services"
)

// ExportData asks the user where to save a backup and writes every task and
// pomodoro session there as JSON. Returns the chosen path, or "" if the user
// cancelled the dialog.
func (a *App) ExportData() (string, error) {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Export TaskMax data",
		DefaultFilename: fmt.Sprintf("taskmax-backup-%s.json", time.Now().Format("2006-01-02")),
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "TaskMax backup (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // dialog cancelled
	}
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		path += ".json"
	}

	data, err := a.backup.Export(version)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ImportData asks the user for a backup file and imports it. Mode "merge"
// appends the backup's tasks to the current board; "replace" wipes the board
// first and restores the backup exactly (confirmed with a native dialog).
func (a *App) ImportData(mode string) (services.ImportResult, error) {
	if mode == services.ImportReplace {
		choice, err := wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
			Type:          wailsruntime.QuestionDialog,
			Title:         "Replace all data?",
			Message:       "This deletes every task and session currently in TaskMax and restores the backup instead. Continue?",
			Buttons:       []string{"Replace", "Cancel"},
			DefaultButton: "Cancel",
		})
		if err != nil {
			return services.ImportResult{}, err
		}
		// Windows maps QuestionDialog buttons to Yes/No regardless of labels.
		if choice != "Replace" && choice != "Yes" {
			return services.ImportResult{Canceled: true}, nil
		}
	}

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Import TaskMax data",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "TaskMax backup (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return services.ImportResult{}, err
	}
	if path == "" {
		return services.ImportResult{Canceled: true}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return services.ImportResult{}, err
	}
	res, err := a.backup.Import(data, mode)
	if err != nil {
		return services.ImportResult{}, err
	}
	// Same event the MCP server uses — the task list refreshes live.
	wailsruntime.EventsEmit(a.ctx, "tasks:changed")
	return res, nil
}
