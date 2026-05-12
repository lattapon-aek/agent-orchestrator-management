package session

import (
	"fmt"
	"testing"
	"time"
)

func TestServiceCreateAndListByProject(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	nextID := 0
	service := NewServiceWithIDGenerator(sqlDB, func() string {
		nextID++
		return fmt.Sprintf("SESS-TEST-%d", nextID)
	})

	record, err := service.Create(CreateParams{
		ProjectID:       "my-app",
		AgentID:         "my-app:backend-main",
		AgentName:       "backend-main",
		RoleName:        "backend",
		Runtime:         "codex",
		RepoPath:        "C:/repo",
		TmuxSessionName: "aom-my-app",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if record.Status != "Created" {
		t.Fatalf("Status = %q, want %q", record.Status, "Created")
	}
	if record.ID != "SESS-TEST-1" {
		t.Fatalf("ID = %q, want %q", record.ID, "SESS-TEST-1")
	}

	loaded, err := service.Get(record.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Get returned nil record")
	}
	if loaded.AgentName != "backend-main" {
		t.Fatalf("AgentName = %q, want %q", loaded.AgentName, "backend-main")
	}

	sessions, err := service.ListByProject("my-app")
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}
}

func TestServiceCreateValidatesRequiredFields(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	service := NewServiceWithIDGenerator(sqlDB, func() string { return "SESS-TEST-1" })

	_, err := service.Create(CreateParams{
		ProjectID: "my-app",
		Runtime:   "codex",
		RepoPath:  "C:/repo",
	})
	if err == nil {
		t.Fatal("Create should fail when agent name and role are missing")
	}
}

func TestServiceReconcileBindingMarksDetachedWhenPaneIsGone(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	service := NewServiceWithIDGenerator(sqlDB, func() string { return "SESS-TEST-1" })

	record, err := service.Create(CreateParams{
		ProjectID:       "my-app",
		AgentID:         "my-app:backend-main",
		AgentName:       "backend-main",
		RoleName:        "backend",
		Runtime:         "codex",
		Status:          "Idle",
		RepoPath:        "C:/repo",
		TmuxSessionName: "aom-my-app",
		TmuxPane:        "%7",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reconciled, err := service.ReconcileBinding(*record, false)
	if err != nil {
		t.Fatalf("ReconcileBinding failed: %v", err)
	}
	if reconciled.Status != "Detached" {
		t.Fatalf("Status = %q, want Detached", reconciled.Status)
	}
}

func TestServiceReconcileBindingRestoresDetachedSessionWhenPaneReturns(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	service := NewServiceWithIDGenerator(sqlDB, func() string { return "SESS-TEST-1" })
	service.now = func() time.Time {
		return time.Date(2026, 5, 12, 21, 30, 0, 0, time.FixedZone("ICT", 7*60*60))
	}

	record, err := service.Create(CreateParams{
		ProjectID:       "my-app",
		AgentID:         "my-app:backend-main",
		AgentName:       "backend-main",
		RoleName:        "backend",
		Runtime:         "codex",
		Status:          "Detached",
		RepoPath:        "C:/repo",
		TmuxSessionName: "aom-my-app",
		TmuxPane:        "%7",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reconciled, err := service.ReconcileBinding(*record, true)
	if err != nil {
		t.Fatalf("ReconcileBinding failed: %v", err)
	}
	if reconciled.Status != "Idle" {
		t.Fatalf("Status = %q, want Idle", reconciled.Status)
	}
	if reconciled.LastSeenAt == nil {
		t.Fatal("LastSeenAt is nil, want timestamp")
	}
}

func TestServiceStopMarksSessionStopped(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	service := NewServiceWithIDGenerator(sqlDB, func() string { return "SESS-TEST-1" })
	record, err := service.Create(CreateParams{
		ProjectID:       "my-app",
		AgentID:         "my-app:backend-main",
		AgentName:       "backend-main",
		RoleName:        "backend",
		Runtime:         "codex",
		Status:          "Idle",
		RepoPath:        "C:/repo",
		TmuxSessionName: "aom-my-app",
		TmuxPane:        "%7",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stopped, err := service.Stop(*record)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if stopped.Status != "Stopped" {
		t.Fatalf("Status = %q, want Stopped", stopped.Status)
	}
}

func TestServiceArchiveMarksStoppedSessionArchived(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	service := NewServiceWithIDGenerator(sqlDB, func() string { return "SESS-TEST-1" })
	record, err := service.Create(CreateParams{
		ProjectID: "my-app",
		AgentName: "backend-main",
		RoleName:  "backend",
		Runtime:   "codex",
		Status:    "Stopped",
		RepoPath:  "C:/repo",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	archived, err := service.Archive(*record)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}
	if archived.Status != "Archived" {
		t.Fatalf("Status = %q, want Archived", archived.Status)
	}
}

func TestServiceArchiveRejectsIdleSession(t *testing.T) {
	sqlDB := openTestDB(t)
	defer sqlDB.Close()

	service := NewServiceWithIDGenerator(sqlDB, func() string { return "SESS-TEST-1" })
	record, err := service.Create(CreateParams{
		ProjectID: "my-app",
		AgentName: "backend-main",
		RoleName:  "backend",
		Runtime:   "codex",
		Status:    "Idle",
		RepoPath:  "C:/repo",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := service.Archive(*record); err == nil {
		t.Fatal("Archive unexpectedly succeeded")
	}
}
