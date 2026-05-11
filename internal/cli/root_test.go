package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteProjectInitCreatesAOMStructure(t *testing.T) {
	repoRoot := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err = Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	requiredPaths := []string{
		filepath.Join(repoRoot, ".aom", "project.yaml"),
		filepath.Join(repoRoot, ".aom", "agents.yaml"),
		filepath.Join(repoRoot, ".aom", "resources.yaml"),
		filepath.Join(repoRoot, ".aom", "policy.yaml"),
		filepath.Join(repoRoot, ".aom", "sessions.db"),
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%q) failed: %v", path, err)
		}
	}

	if got := stdout.String(); !strings.Contains(got, "Project initialized") {
		t.Fatalf("stdout = %q, want project initialized message", got)
	}
}

func TestExecuteOpenShowsProjectSummary(t *testing.T) {
	repoRoot := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	if err := Execute([]string{"open"}, &stdout, &stderr); err != nil {
		t.Fatalf("open failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Project opened") {
		t.Fatalf("stdout = %q, want Project opened", out)
	}
	if !strings.Contains(out, "backend-main") {
		t.Fatalf("stdout = %q, want backend-main in summary", out)
	}
}

func TestExecuteStatusShowsProjectSummary(t *testing.T) {
	repoRoot := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Project status") {
		t.Fatalf("stdout = %q, want Project status", out)
	}
	if !strings.Contains(out, "Agents:") {
		t.Fatalf("stdout = %q, want Agents section", out)
	}
}
