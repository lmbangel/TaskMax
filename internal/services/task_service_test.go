package services

import (
	"testing"
	"time"

	"taskmax/internal/models"
)

func TestCreateAppliesDefaults(t *testing.T) {
	s := NewTaskService(newTestDB(t))

	created := mustCreateTask(t, s, models.Task{Title: "write tests"})
	if created.Status != "todo" {
		t.Errorf("default status = %q, want todo", created.Status)
	}
	if created.Priority != "medium" {
		t.Errorf("default priority = %q, want medium", created.Priority)
	}
}

func TestCreateRequiresTitle(t *testing.T) {
	s := NewTaskService(newTestDB(t))
	if _, err := s.Create(models.Task{}); err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestCreateSortsNewTasksToTop(t *testing.T) {
	s := NewTaskService(newTestDB(t))

	first := mustCreateTask(t, s, models.Task{Title: "first"})
	second := mustCreateTask(t, s, models.Task{Title: "second"})
	if second.Position >= first.Position {
		t.Errorf("second.Position = %d, want < first.Position (%d)", second.Position, first.Position)
	}

	all, err := s.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if all[0].Title != "second" {
		t.Errorf("newest task should list first, got %q", all[0].Title)
	}
}

func TestUpdateClearsDueDate(t *testing.T) {
	s := NewTaskService(newTestDB(t))

	due := time.Now().AddDate(0, 0, 1)
	created := mustCreateTask(t, s, models.Task{Title: "dated", DueDate: &due})

	created.DueDate = nil
	updated, err := s.Update(created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DueDate != nil {
		t.Errorf("due date should be cleared, got %v", updated.DueDate)
	}
}

func TestCompletingRecurringTaskSpringsBack(t *testing.T) {
	s := NewTaskService(newTestDB(t))

	yesterday := time.Now().AddDate(0, 0, -1)
	created := mustCreateTask(t, s, models.Task{
		Title:      "water plants",
		Recurrence: "daily",
		DueDate:    &yesterday,
	})

	created.Status = "done"
	updated, err := s.Update(created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "todo" {
		t.Errorf("recurring task status = %q, want todo", updated.Status)
	}
	if updated.DueDate == nil || !updated.DueDate.After(time.Now()) {
		t.Errorf("recurring due date should advance into the future, got %v", updated.DueDate)
	}
}

func TestCompletingNonRecurringTaskStaysDone(t *testing.T) {
	s := NewTaskService(newTestDB(t))

	created := mustCreateTask(t, s, models.Task{Title: "one-off"})
	created.Status = "done"
	updated, err := s.Update(created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("status = %q, want done", updated.Status)
	}
}

func TestNextOccurrence(t *testing.T) {
	now := time.Now()
	past := now.AddDate(0, 0, -10)

	cases := []struct {
		name       string
		due        *time.Time
		recurrence string
	}{
		{"daily from past", &past, "daily"},
		{"weekly from past", &past, "weekly"},
		{"monthly from past", &past, "monthly"},
		{"daily with no due date", nil, "daily"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := nextOccurrence(tc.due, tc.recurrence)
			if !next.After(now) {
				t.Errorf("nextOccurrence = %v, want after now", next)
			}
		})
	}

	// A far-future due date should not be advanced further.
	future := now.AddDate(0, 0, 30)
	if got := nextOccurrence(&future, "daily"); !got.Equal(future) {
		t.Errorf("future due date moved from %v to %v", future, got)
	}
}

func TestSetInProgressIfTodo(t *testing.T) {
	db := newTestDB(t)
	s := NewTaskService(db)

	todo := mustCreateTask(t, s, models.Task{Title: "todo task"})
	done := mustCreateTask(t, s, models.Task{Title: "done task", Status: "done"})

	if err := s.SetInProgressIfTodo(todo.ID); err != nil {
		t.Fatalf("SetInProgressIfTodo: %v", err)
	}
	if err := s.SetInProgressIfTodo(done.ID); err != nil {
		t.Fatalf("SetInProgressIfTodo: %v", err)
	}

	// Separate structs: GORM folds a previous result's primary key into the
	// next query's conditions if the destination is reused.
	var reloadedTodo, reloadedDone models.Task
	db.First(&reloadedTodo, todo.ID)
	if reloadedTodo.Status != "in_progress" {
		t.Errorf("todo task status = %q, want in_progress", reloadedTodo.Status)
	}
	db.First(&reloadedDone, done.ID)
	if reloadedDone.Status != "done" {
		t.Errorf("done task status = %q, want done (unchanged)", reloadedDone.Status)
	}
}

func TestDeleteCascadesSessions(t *testing.T) {
	db := newTestDB(t)
	s := NewTaskService(db)

	task := mustCreateTask(t, s, models.Task{Title: "with history"})
	db.Create(&models.PomodoroSession{TaskID: task.ID, Type: SessionWork, Duration: 25, StartedAt: time.Now()})

	if err := s.Delete(task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var sessions int64
	db.Model(&models.PomodoroSession{}).Where("task_id = ?", task.ID).Count(&sessions)
	if sessions != 0 {
		t.Errorf("sessions left after task delete = %d, want 0", sessions)
	}
}

func TestReorder(t *testing.T) {
	s := NewTaskService(newTestDB(t))

	a := mustCreateTask(t, s, models.Task{Title: "a"})
	b := mustCreateTask(t, s, models.Task{Title: "b"})
	c := mustCreateTask(t, s, models.Task{Title: "c"})

	if err := s.Reorder([]uint{c.ID, a.ID, b.ID}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	all, err := s.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	want := []string{"c", "a", "b"}
	for i, task := range all {
		if task.Title != want[i] {
			t.Errorf("position %d = %q, want %q", i, task.Title, want[i])
		}
	}
}
