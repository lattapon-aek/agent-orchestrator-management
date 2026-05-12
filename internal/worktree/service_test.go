package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/db"
)

func TestServiceCreatePlannedCreatesMapping(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer sqlDB.Close()

	service := NewService(sqlDB)

	record, err := service.CreatePlanned(CreateParams{
		ProjectID:     "proj-1",
		TaskID:        "TASK-001",
		TaskTitle:     "Fix login validation",
		RepoPath:      "C:/repo",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreatePlanned failed: %v", err)
	}

	if record.Status != "Planned" {
		t.Fatalf("Status = %q, want Planned", record.Status)
	}
	if record.BaseBranch != "main" {
		t.Fatalf("BaseBranch = %q, want main", record.BaseBranch)
	}
	if !strings.HasPrefix(record.BranchName, "aom/task-001-fix-login-validation") {
		t.Fatalf("BranchName = %q, want sanitized branch prefix", record.BranchName)
	}
	if !strings.Contains(record.WorktreePath, filepath.Join(".aom", "worktrees")) {
		t.Fatalf("WorktreePath = %q, want .aom/worktrees path", record.WorktreePath)
	}

	loaded, err := service.GetByTask("TASK-001")
	if err != nil {
		t.Fatalf("GetByTask failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("GetByTask returned nil record")
	}
	if loaded.TaskID != "TASK-001" {
		t.Fatalf("TaskID = %q, want TASK-001", loaded.TaskID)
	}
}

func TestServiceEnsureProvisionedSkipsNonGitRepo(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer sqlDB.Close()

	service := NewService(sqlDB)
	service.lookPath = func(string) (string, error) { return "git", nil }
	service.runGit = func(string, ...string) ([]byte, error) { return nil, fmt.Errorf("not a git repo") }

	record, err := service.CreatePlanned(CreateParams{
		ProjectID:     "proj-1",
		TaskID:        "TASK-001",
		TaskTitle:     "Fix login validation",
		RepoPath:      "C:/repo",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreatePlanned failed: %v", err)
	}

	provisioned, err := service.EnsureProvisioned(record.TaskID, "C:/repo")
	if err != nil {
		t.Fatalf("EnsureProvisioned failed: %v", err)
	}
	if provisioned.Status != "Planned" {
		t.Fatalf("Status = %q, want Planned", provisioned.Status)
	}
}

func TestServiceEnsureProvisionedMarksReadyAfterGitWorktreeAdd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer sqlDB.Close()

	repoRoot := t.TempDir()
	service := NewService(sqlDB)
	service.lookPath = func(string) (string, error) { return "git", nil }
	service.stat = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	service.mkdirAll = func(string, os.FileMode) error { return nil }
	var calls [][]string
	service.runGit = func(repoPath string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{repoPath}, args...))
		if len(args) >= 2 && args[0] == "rev-parse" {
			return []byte("true\n"), nil
		}
		if len(args) >= 2 && args[0] == "worktree" && args[1] == "add" {
			return []byte("prepared\n"), nil
		}
		return nil, nil
	}

	record, err := service.CreatePlanned(CreateParams{
		ProjectID:     "proj-1",
		TaskID:        "TASK-001",
		TaskTitle:     "Fix login validation",
		RepoPath:      repoRoot,
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreatePlanned failed: %v", err)
	}

	provisioned, err := service.EnsureProvisioned(record.TaskID, repoRoot)
	if err != nil {
		t.Fatalf("EnsureProvisioned failed: %v", err)
	}
	if provisioned.Status != "Ready" {
		t.Fatalf("Status = %q, want Ready", provisioned.Status)
	}
	if len(calls) < 2 {
		t.Fatalf("git calls = %d, want at least 2", len(calls))
	}
	if !strings.Contains(strings.Join(calls[1], " "), "worktree add -b") {
		t.Fatalf("calls[1] = %#v, want worktree add", calls[1])
	}
}
