package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/step"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/task"
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
