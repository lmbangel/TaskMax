package services

import (
	"errors"

	"gorm.io/gorm"

	"taskmax/internal/models"
)

// TaskService encapsulates all task persistence logic.
type TaskService struct {
	db *gorm.DB
}

// NewTaskService constructs a TaskService backed by the given DB.
func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{db: db}
}

// GetAll returns every task ordered by manual position, then newest first.
func (s *TaskService) GetAll() ([]models.Task, error) {
	var tasks []models.Task
	err := s.db.Order("position asc").Order("created_at desc").Find(&tasks).Error
	return tasks, err
}

// GetByStatus returns tasks filtered by status ("todo", "in_progress", "done").
func (s *TaskService) GetByStatus(status string) ([]models.Task, error) {
	var tasks []models.Task
	err := s.db.Where("status = ?", status).
		Order("position asc").Order("created_at desc").
		Find(&tasks).Error
	return tasks, err
}

// Create inserts a new task, applying defaults for empty fields.
func (s *TaskService) Create(task models.Task) (models.Task, error) {
	if task.Title == "" {
		return models.Task{}, errors.New("task title is required")
	}
	if task.Status == "" {
		task.Status = "todo"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	// New tasks sort to the top of the list.
	var maxPos int
	s.db.Model(&models.Task{}).Select("COALESCE(MIN(position), 0)").Scan(&maxPos)
	task.Position = maxPos - 1

	if err := s.db.Create(&task).Error; err != nil {
		return models.Task{}, err
	}
	return task, nil
}

// Update saves changes to an existing task. Select all fields so that clearing
// a value (e.g. removing a due date) is persisted.
func (s *TaskService) Update(task models.Task) (models.Task, error) {
	if task.ID == 0 {
		return models.Task{}, errors.New("task id is required for update")
	}
	err := s.db.Model(&models.Task{}).
		Where("id = ?", task.ID).
		Select("title", "description", "priority", "status", "tags", "due_date", "position").
		Updates(map[string]interface{}{
			"title":       task.Title,
			"description": task.Description,
			"priority":    task.Priority,
			"status":      task.Status,
			"tags":        task.Tags,
			"due_date":    task.DueDate,
			"position":    task.Position,
		}).Error
	if err != nil {
		return models.Task{}, err
	}

	var updated models.Task
	if err := s.db.First(&updated, task.ID).Error; err != nil {
		return models.Task{}, err
	}
	return updated, nil
}

// Delete removes a task and its associated pomodoro sessions.
func (s *TaskService) Delete(id uint) error {
	if id == 0 {
		return errors.New("task id is required for delete")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", id).Delete(&models.PomodoroSession{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Task{}, id).Error
	})
}

// Reorder persists a new ordering given a slice of task IDs in display order.
func (s *TaskService) Reorder(orderedIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			if err := tx.Model(&models.Task{}).
				Where("id = ?", id).
				Update("position", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// IncrementPomodoroCount bumps the completed work-session counter on a task.
func (s *TaskService) IncrementPomodoroCount(taskID uint) error {
	if taskID == 0 {
		return nil
	}
	return s.db.Model(&models.Task{}).
		Where("id = ?", taskID).
		UpdateColumn("pomodoro_count", gorm.Expr("pomodoro_count + 1")).Error
}
