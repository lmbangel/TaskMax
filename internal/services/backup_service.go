package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"taskmax/internal/models"
)

// Export file format identifiers. Version is bumped when the payload shape
// changes in a way older builds cannot read.
const (
	exportFormat  = "taskmax-export"
	exportVersion = 1
)

// Import modes.
const (
	ImportMerge   = "merge"   // append the backup's tasks to the current board
	ImportReplace = "replace" // wipe the board and restore the backup exactly
)

// ExportPayload is the on-disk JSON shape of a TaskMax backup.
type ExportPayload struct {
	Format     string                   `json:"format"`
	Version    int                      `json:"version"`
	ExportedAt time.Time                `json:"exported_at"`
	AppVersion string                   `json:"app_version"`
	Tasks      []models.Task            `json:"tasks"`
	Sessions   []models.PomodoroSession `json:"sessions"`
}

// ImportResult reports what an import actually did.
type ImportResult struct {
	Canceled         bool `json:"canceled"`
	TasksImported    int  `json:"tasks_imported"`
	SessionsImported int  `json:"sessions_imported"`
}

// BackupService serialises the whole board (tasks + pomodoro history) to JSON
// and restores it, either merging into or replacing the current data.
type BackupService struct {
	db *gorm.DB
}

// NewBackupService constructs a BackupService backed by the given DB.
func NewBackupService(db *gorm.DB) *BackupService {
	return &BackupService{db: db}
}

// Export marshals every task and pomodoro session into a versioned JSON
// document. Tasks keep their display order so an import preserves it.
func (s *BackupService) Export(appVersion string) ([]byte, error) {
	var tasks []models.Task
	if err := s.db.Order("position asc").Order("created_at desc").Find(&tasks).Error; err != nil {
		return nil, err
	}
	var sessions []models.PomodoroSession
	if err := s.db.Order("started_at asc").Find(&sessions).Error; err != nil {
		return nil, err
	}
	payload := ExportPayload{
		Format:     exportFormat,
		Version:    exportVersion,
		ExportedAt: time.Now(),
		AppVersion: appVersion,
		Tasks:      tasks,
		Sessions:   sessions,
	}
	return json.MarshalIndent(payload, "", "  ")
}

// Import restores a backup produced by Export.
//
//   - ImportMerge appends the backup's tasks below the existing board (IDs are
//     remapped, so the same file can be imported into any database).
//   - ImportReplace deletes all current tasks and sessions first and restores
//     the backup exactly, original IDs included.
func (s *BackupService) Import(data []byte, mode string) (ImportResult, error) {
	var payload ExportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return ImportResult{}, fmt.Errorf("not a valid TaskMax export: %w", err)
	}
	if payload.Format != exportFormat {
		return ImportResult{}, errors.New("not a TaskMax export file")
	}
	if payload.Version > exportVersion {
		return ImportResult{}, fmt.Errorf("export version %d is newer than this app supports — update TaskMax first", payload.Version)
	}

	switch mode {
	case ImportReplace:
		return s.importReplace(payload)
	case ImportMerge, "":
		return s.importMerge(payload)
	default:
		return ImportResult{}, fmt.Errorf("unknown import mode %q", mode)
	}
}

func (s *BackupService) importMerge(payload ExportPayload) (ImportResult, error) {
	res := ImportResult{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Imported tasks slot in below everything already on the board.
		var maxPos int
		tx.Model(&models.Task{}).Select("COALESCE(MAX(position), 0)").Scan(&maxPos)

		idMap := make(map[uint]uint, len(payload.Tasks))
		for i := range payload.Tasks {
			src := payload.Tasks[i]
			oldID := src.ID
			src.ID = 0
			src.Position = maxPos + 1 + i
			if err := tx.Create(&src).Error; err != nil {
				return err
			}
			idMap[oldID] = src.ID
			res.TasksImported++
		}
		for i := range payload.Sessions {
			sess := payload.Sessions[i]
			sess.ID = 0
			// Sessions follow their task's new ID; orphans (task since
			// deleted) keep TaskID 0 so they still count in the heatmap.
			sess.TaskID = idMap[sess.TaskID]
			if err := tx.Create(&sess).Error; err != nil {
				return err
			}
			res.SessionsImported++
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	return res, nil
}

func (s *BackupService) importReplace(payload ExportPayload) (ImportResult, error) {
	res := ImportResult{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		wipe := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()
		if err := wipe.Delete(&models.PomodoroSession{}).Error; err != nil {
			return err
		}
		if err := wipe.Delete(&models.Task{}).Error; err != nil {
			return err
		}
		for i := range payload.Tasks {
			if err := tx.Create(&payload.Tasks[i]).Error; err != nil {
				return err
			}
			res.TasksImported++
		}
		for i := range payload.Sessions {
			if err := tx.Create(&payload.Sessions[i]).Error; err != nil {
				return err
			}
			res.SessionsImported++
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	return res, nil
}
