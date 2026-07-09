package services

import (
	"testing"
	"time"

	"taskmax/internal/models"
)

func TestAddAndListComments(t *testing.T) {
	s := NewTaskService(newTestDB(t))
	task := mustCreateTask(t, s, models.Task{Title: "commented"})

	first, err := s.AddComment(task.ID, "started looking at this", "", "")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if first.Source != "" || first.Author != "" {
		t.Errorf("user comment source/author = %q/%q, want empty", first.Source, first.Author)
	}

	agent, err := s.AddComment(task.ID, "implemented and tested, see PR #42", "agent", "Claude Code")
	if err != nil {
		t.Fatalf("AddComment (agent): %v", err)
	}
	if agent.Source != "agent" || agent.Author != "Claude Code" {
		t.Errorf("agent comment source/author = %q/%q", agent.Source, agent.Author)
	}

	comments, err := s.CommentsForTask(task.ID)
	if err != nil {
		t.Fatalf("CommentsForTask: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("comment count = %d, want 2", len(comments))
	}
	// Oldest first, so the trail reads chronologically.
	if comments[0].Body != "started looking at this" {
		t.Errorf("first comment = %q, want the earliest", comments[0].Body)
	}
}

func TestAddCommentValidation(t *testing.T) {
	s := NewTaskService(newTestDB(t))
	task := mustCreateTask(t, s, models.Task{Title: "t"})

	if _, err := s.AddComment(task.ID, "", "", ""); err == nil {
		t.Error("expected error for empty body")
	}
	if _, err := s.AddComment(0, "hi", "", ""); err == nil {
		t.Error("expected error for missing task id")
	}
	if _, err := s.AddComment(9999, "hi", "", ""); err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestDeleteComment(t *testing.T) {
	s := NewTaskService(newTestDB(t))
	task := mustCreateTask(t, s, models.Task{Title: "t"})
	c, _ := s.AddComment(task.ID, "to be removed", "", "")

	if err := s.DeleteComment(c.ID); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	comments, _ := s.CommentsForTask(task.ID)
	if len(comments) != 0 {
		t.Errorf("comments after delete = %d, want 0", len(comments))
	}
}

func TestDeleteTaskCascadesComments(t *testing.T) {
	db := newTestDB(t)
	s := NewTaskService(db)
	task := mustCreateTask(t, s, models.Task{Title: "with comments"})
	s.AddComment(task.ID, "one", "", "")
	s.AddComment(task.ID, "two", "agent", "")

	if err := s.Delete(task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Comment{}).Where("task_id = ?", task.ID).Count(&count)
	if count != 0 {
		t.Errorf("comments left after task delete = %d, want 0", count)
	}
}

func TestBackupRoundTripIncludesComments(t *testing.T) {
	srcDB := newTestDB(t)
	src := NewTaskService(srcDB)
	task := mustCreateTask(t, src, models.Task{Title: "traced"})
	src.AddComment(task.ID, "agent did the thing", "agent", "Claude Code")
	srcDB.Create(&models.PomodoroSession{TaskID: task.ID, Type: SessionWork, Duration: 25, Completed: true, StartedAt: time.Now()})

	data, err := NewBackupService(srcDB).Export("test")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	dstDB := newTestDB(t)
	res, err := NewBackupService(dstDB).Import(data, ImportMerge)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.CommentsImported != 1 {
		t.Errorf("CommentsImported = %d, want 1", res.CommentsImported)
	}

	var imported models.Task
	dstDB.Where("title = ?", "traced").First(&imported)
	comments, _ := NewTaskService(dstDB).CommentsForTask(imported.ID)
	if len(comments) != 1 || comments[0].Author != "Claude Code" {
		t.Errorf("imported comments = %+v, want the agent comment reattached", comments)
	}
}
