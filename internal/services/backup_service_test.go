package services

import (
	"testing"
	"time"

	"taskmax/internal/models"
)

func TestExportImportMergeRoundTrip(t *testing.T) {
	srcDB := newTestDB(t)
	src := NewTaskService(srcDB)
	backup := NewBackupService(srcDB)

	taskA := mustCreateTask(t, src, models.Task{Title: "alpha", Tags: "one,two"})
	mustCreateTask(t, src, models.Task{Title: "beta", Status: "done"})
	srcDB.Create(&models.PomodoroSession{TaskID: taskA.ID, Type: SessionWork, Duration: 25, Completed: true, StartedAt: time.Now()})

	data, err := backup.Export("test")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Import into a database that already has a task of its own.
	dstDB := newTestDB(t)
	dst := NewTaskService(dstDB)
	existing := mustCreateTask(t, dst, models.Task{Title: "pre-existing"})

	res, err := NewBackupService(dstDB).Import(data, ImportMerge)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.TasksImported != 2 || res.SessionsImported != 1 {
		t.Errorf("imported %d tasks / %d sessions, want 2 / 1", res.TasksImported, res.SessionsImported)
	}

	all, _ := dst.GetAll()
	if len(all) != 3 {
		t.Fatalf("board has %d tasks after merge, want 3", len(all))
	}

	// The imported session must point at the *new* ID of "alpha".
	var imported models.Task
	dstDB.Where("title = ?", "alpha").First(&imported)
	if imported.ID == existing.ID {
		t.Fatal("imported task reused an existing ID")
	}
	var sessions int64
	dstDB.Model(&models.PomodoroSession{}).Where("task_id = ?", imported.ID).Count(&sessions)
	if sessions != 1 {
		t.Errorf("sessions attached to imported task = %d, want 1", sessions)
	}
}

func TestImportReplaceWipesBoardFirst(t *testing.T) {
	srcDB := newTestDB(t)
	src := NewTaskService(srcDB)
	taskA := mustCreateTask(t, src, models.Task{Title: "from backup"})
	srcDB.Create(&models.PomodoroSession{TaskID: taskA.ID, Type: SessionWork, Duration: 25, Completed: true, StartedAt: time.Now()})

	data, err := NewBackupService(srcDB).Export("test")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	dstDB := newTestDB(t)
	dst := NewTaskService(dstDB)
	doomed := mustCreateTask(t, dst, models.Task{Title: "doomed"})
	dstDB.Create(&models.PomodoroSession{TaskID: doomed.ID, Type: SessionWork, Duration: 25, StartedAt: time.Now()})

	if _, err := NewBackupService(dstDB).Import(data, ImportReplace); err != nil {
		t.Fatalf("Import: %v", err)
	}

	all, _ := dst.GetAll()
	if len(all) != 1 || all[0].Title != "from backup" {
		t.Fatalf("board after replace = %+v, want exactly the backup's task", all)
	}
	// Original IDs survive a replace.
	if all[0].ID != taskA.ID {
		t.Errorf("restored task ID = %d, want %d", all[0].ID, taskA.ID)
	}
	var sessions int64
	dstDB.Model(&models.PomodoroSession{}).Count(&sessions)
	if sessions != 1 {
		t.Errorf("session count after replace = %d, want 1", sessions)
	}
}

func TestImportRejectsGarbage(t *testing.T) {
	backup := NewBackupService(newTestDB(t))

	if _, err := backup.Import([]byte("not json"), ImportMerge); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if _, err := backup.Import([]byte(`{"format":"something-else","version":1}`), ImportMerge); err == nil {
		t.Error("expected error for wrong format marker")
	}
	if _, err := backup.Import([]byte(`{"format":"taskmax-export","version":999}`), ImportMerge); err == nil {
		t.Error("expected error for too-new export version")
	}
	if _, err := backup.Import([]byte(`{"format":"taskmax-export","version":1}`), "sideways"); err == nil {
		t.Error("expected error for unknown import mode")
	}
}
