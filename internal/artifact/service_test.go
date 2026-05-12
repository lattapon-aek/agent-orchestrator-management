package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/step"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/task"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/worktree"
)

func TestServiceSeedTaskArtifactsCreatesCoreFiles(t *testing.T) {
	repoRoot := t.TempDir()
	service := NewService(repoRoot, "tasks")
	service.now = func() time.Time { return time.Date(2026, 5, 11, 21, 0, 0, 0, time.FixedZone("ICT", 7*60*60)) }
	service.eventIDGen = func() string { return "EVT-001" }

	err := service.SeedTaskArtifacts(SyncParams{
		Task: task.Record{
			ID:             "TASK-001",
			Title:          "Fix login validation",
			Mode:           "Direct",
			Status:         "Planned",
			PreferredRole:  "backend",
			PreferredAgent: "backend-main",
		},
		Steps: []step.Record{
			{
				ID:        "STEP-001",
				TaskID:    "TASK-001",
				StepType:  "implementation",
				Title:     "Fix login validation",
				Status:    "Proposed",
				RoleName:  "backend",
				AgentName: "backend-main",
			},
		},
		Worktree: &worktree.Record{
			TaskID:       "TASK-001",
			ProjectID:    "proj-1",
			Status:       "Planned",
			BaseBranch:   "main",
			BranchName:   "aom/task-001-fix-login-validation",
			WorktreePath: filepath.Join(repoRoot, ".aom", "worktrees", "task-001-fix-login-validation"),
		},
		CreatedBy:             "operator",
		UpdatedBy:             "operator",
		RecommendedNextAction: "confirm the proposed step and move the task to Ready",
	})
	if err != nil {
		t.Fatalf("SeedTaskArtifacts failed: %v", err)
	}

	taskDir := filepath.Join(repoRoot, ".aom", "tasks", "TASK-001")
	for _, name := range []string{"task.md", "state.md", "index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join(taskDir, name)); err != nil {
			t.Fatalf("artifact %s missing: %v", name, err)
		}
	}

	taskData, err := os.ReadFile(filepath.Join(taskDir, "task.md"))
	if err != nil {
		t.Fatalf("ReadFile(task.md) failed: %v", err)
	}
	if !strings.Contains(string(taskData), "Worktree: "+filepath.Join(repoRoot, ".aom", "worktrees", "task-001-fix-login-validation")) {
		t.Fatalf("task.md = %q, want mapped worktree path", string(taskData))
	}

	indexData, err := os.ReadFile(filepath.Join(taskDir, "index.md"))
	if err != nil {
		t.Fatalf("ReadFile(index.md) failed: %v", err)
	}
	if !strings.Contains(string(indexData), "Worktree Status: Planned") {
		t.Fatalf("index.md = %q, want planned worktree status", string(indexData))
	}

	logData, err := os.ReadFile(filepath.Join(taskDir, "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "task.created") {
		t.Fatalf("log.md = %q, want task.created event", string(logData))
	}
	if !strings.Contains(string(logData), "step.proposed") {
		t.Fatalf("log.md = %q, want step.proposed event", string(logData))
	}
}

func TestServiceSeedTaskArtifactsWritesIntoReadyWorktreeAgentDir(t *testing.T) {
	repoRoot := t.TempDir()
	worktreePath := filepath.Join(repoRoot, ".aom", "worktrees", "task-003-fix-login-validation")
	service := NewService(repoRoot, "tasks")

	err := service.SeedTaskArtifacts(SyncParams{
		Task: task.Record{
			ID:             "TASK-003",
			Title:          "Fix login validation",
			Mode:           "Direct",
			Status:         "Planned",
			PreferredRole:  "backend",
			PreferredAgent: "backend-main",
		},
		Steps: []step.Record{
			{ID: "STEP-001", TaskID: "TASK-003", StepType: "implementation", Title: "Fix login validation", Status: "Proposed"},
		},
		Worktree: &worktree.Record{
			TaskID:       "TASK-003",
			ProjectID:    "proj-1",
			Status:       "Ready",
			BaseBranch:   "main",
			BranchName:   "aom/task-003-fix-login-validation",
			WorktreePath: worktreePath,
		},
		CreatedBy:             "operator",
		UpdatedBy:             "operator",
		RecommendedNextAction: "confirm the proposed step and move the task to Ready",
	})
	if err != nil {
		t.Fatalf("SeedTaskArtifacts failed: %v", err)
	}

	agentDir := filepath.Join(worktreePath, ".agent")
	for _, name := range []string{"task.md", "state.md", "index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join(agentDir, name)); err != nil {
			t.Fatalf("artifact %s missing: %v", name, err)
		}
	}

	if _, err := os.Stat(filepath.Join(repoRoot, ".aom", "tasks", "TASK-003", "task.md")); !os.IsNotExist(err) {
		t.Fatalf("fallback artifact unexpectedly exists: %v", err)
	}

	taskData, err := os.ReadFile(filepath.Join(agentDir, "task.md"))
	if err != nil {
		t.Fatalf("ReadFile(task.md) failed: %v", err)
	}
	if !strings.Contains(string(taskData), "Artifact Root: "+filepath.Join(worktreePath, ".agent")) {
		t.Fatalf("task.md = %q, want .agent artifact root", string(taskData))
	}
}

func TestServiceSeedTaskArtifactsCreatesModeSpecificFiles(t *testing.T) {
	repoRoot := t.TempDir()
	service := NewService(repoRoot, "tasks")

	err := service.SeedTaskArtifacts(SyncParams{
		Task: task.Record{
			ID:             "TASK-002",
			Title:          "Capture checkout requirements",
			Mode:           "Requirements-first",
			Status:         "Planned",
			PreferredRole:  "backend",
			PreferredAgent: "backend-main",
		},
		Steps: []step.Record{
			{ID: "STEP-001", TaskID: "TASK-002", StepType: "research", Title: "Capture requirements", Status: "Proposed"},
			{ID: "STEP-002", TaskID: "TASK-002", StepType: "coordination", Title: "Turn accepted requirements into implementation steps", Status: "Proposed", Dependencies: []string{"STEP-001"}},
		},
		CreatedBy:             "operator",
		UpdatedBy:             "operator",
		RecommendedNextAction: "confirm the requirements-first mode, then create the task and capture the first requirement step",
	})
	if err != nil {
		t.Fatalf("SeedTaskArtifacts failed: %v", err)
	}

	taskDir := filepath.Join(repoRoot, ".aom", "tasks", "TASK-002")
	for _, name := range []string{"requirements.md", "tasks.md"} {
		if _, err := os.Stat(filepath.Join(taskDir, name)); err != nil {
			t.Fatalf("mode artifact %s missing: %v", name, err)
		}
	}
}
