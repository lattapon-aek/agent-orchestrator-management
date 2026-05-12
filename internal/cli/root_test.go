package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/app"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/project"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/tmux"
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

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
	if !strings.Contains(out, "Terminal:") {
		t.Fatalf("stdout = %q, want Terminal section", out)
	}
	if !strings.Contains(out, "Workspace: aom-my-app") {
		t.Fatalf("stdout = %q, want workspace summary", out)
	}
	if !strings.Contains(out, "Workspace state: reused") {
		t.Fatalf("stdout = %q, want workspace state", out)
	}
	if !strings.Contains(out, "Sessions:") {
		t.Fatalf("stdout = %q, want Sessions section", out)
	}
	if !strings.Contains(out, "  None") {
		t.Fatalf("stdout = %q, want no sessions placeholder", out)
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "", errors.New("not found") },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

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
	if !strings.Contains(out, "Terminal:") {
		t.Fatalf("stdout = %q, want Terminal section", out)
	}
	if !strings.Contains(out, "Sessions:") {
		t.Fatalf("stdout = %q, want Sessions section", out)
	}
}

func TestExecuteSessionSpawnAndList(t *testing.T) {
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

	firstHasSession := true
	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(name string, args ...string) ([]byte, error) {
			if len(args) == 0 {
				return nil, nil
			}

			switch args[0] {
			case "has-session":
				if firstHasSession {
					firstHasSession = false
					return nil, errors.New("session not found")
				}
				return nil, nil
			case "new-session":
				return nil, nil
			case "split-window":
				return []byte("@1 %5\n"), nil
			case "set-option":
				return nil, nil
			default:
				return nil, nil
			}
		},
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	if err := Execute([]string{"session", "spawn", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("session spawn failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Session spawned") {
		t.Fatalf("stdout = %q, want Session spawned", out)
	}
	if !strings.Contains(out, "Pane: %5") {
		t.Fatalf("stdout = %q, want pane binding", out)
	}
	if !strings.Contains(out, "Launch mode: placeholder") {
		t.Fatalf("stdout = %q, want placeholder launch mode", out)
	}

	stdout.Reset()
	stderr.Reset()

	if err := Execute([]string{"session", "list"}, &stdout, &stderr); err != nil {
		t.Fatalf("session list failed: %v", err)
	}

	out = stdout.String()
	if !strings.Contains(out, "Sessions") {
		t.Fatalf("stdout = %q, want Sessions header", out)
	}
	if !strings.Contains(out, "agent=backend-main") {
		t.Fatalf("stdout = %q, want agent listing", out)
	}
	if !strings.Contains(out, "tmux=aom-my-app @1 %5") {
		t.Fatalf("stdout = %q, want tmux binding", out)
	}
}

func TestExecuteSessionSpawnWithMockRuntime(t *testing.T) {
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

	firstHasSession := true
	var splitCommands []string
	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(name string, args ...string) ([]byte, error) {
			if len(args) == 0 {
				return nil, nil
			}

			switch args[0] {
			case "has-session":
				if firstHasSession {
					firstHasSession = false
					return nil, errors.New("session not found")
				}
				return nil, nil
			case "new-session":
				return nil, nil
			case "split-window":
				splitCommands = append(splitCommands, args[len(args)-1])
				return []byte("@1 %5\n"), nil
			case "set-option":
				return nil, nil
			default:
				return nil, nil
			}
		},
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	if err := Execute([]string{"session", "spawn", "backend-main", "--mock"}, &stdout, &stderr); err != nil {
		t.Fatalf("session spawn failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Launch mode: mock") {
		t.Fatalf("stdout = %q, want mock launch mode", out)
	}
	if len(splitCommands) != 1 {
		t.Fatalf("len(splitCommands) = %d, want 1", len(splitCommands))
	}
	if !strings.Contains(splitCommands[0], "AOM mock runtime boot") {
		t.Fatalf("split command = %q, want mock runtime transcript", splitCommands[0])
	}
}

func TestExecuteSessionSpawnWithTaskRefreshesArtifacts(t *testing.T) {
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

	firstHasSession := true
	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(name string, args ...string) ([]byte, error) {
			if len(args) == 0 {
				return nil, nil
			}
			switch args[0] {
			case "has-session":
				if firstHasSession {
					firstHasSession = false
					return nil, errors.New("session not found")
				}
				return nil, nil
			case "new-session":
				return nil, nil
			case "split-window":
				return []byte("@1 %5\n"), nil
			case "set-option":
				return nil, nil
			default:
				return nil, nil
			}
		},
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	if err := Execute([]string{"task", "create", "Bind session to task", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "spawn", "backend-main", "--task", taskID, "--mock"}, &stdout, &stderr); err != nil {
		t.Fatalf("session spawn failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Task: "+taskID) {
		t.Fatalf("stdout = %q, want task in spawn output", out)
	}
	sessionID := extractSessionID(out)
	if sessionID == "" {
		t.Fatalf("could not extract session id from %q", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "list"}, &stdout, &stderr); err != nil {
		t.Fatalf("session list failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "task="+taskID) {
		t.Fatalf("stdout = %q, want task id in session list", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Task: "+taskID) {
		t.Fatalf("stdout = %q, want task in session show", out)
	}

	indexData, err := os.ReadFile(filepath.Join(repoRoot, ".aom", "tasks", taskID, "index.md"))
	if err != nil {
		t.Fatalf("ReadFile(index.md) failed: %v", err)
	}
	if !strings.Contains(string(indexData), "Active Session: "+sessionID) {
		t.Fatalf("index.md = %q, want active session", string(indexData))
	}

	logData, err := os.ReadFile(filepath.Join(repoRoot, ".aom", "tasks", taskID, "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "session.created") {
		t.Fatalf("log.md = %q, want session.created event", string(logData))
	}
}

func TestExecuteSessionShowAttachAndCapture(t *testing.T) {
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

	firstHasSession := true
	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(name string, args ...string) ([]byte, error) {
			if len(args) == 0 {
				return nil, nil
			}
			switch args[0] {
			case "has-session":
				if firstHasSession {
					firstHasSession = false
					return nil, errors.New("session not found")
				}
				return nil, nil
			case "new-session":
				return nil, nil
			case "split-window":
				return []byte("@1 %5\n"), nil
			case "set-option":
				return nil, nil
			case "select-pane":
				return nil, nil
			case "capture-pane":
				return []byte("hello from pane\n"), nil
			default:
				return nil, nil
			}
		},
		func(name string, args ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "spawn", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("session spawn failed: %v", err)
	}

	spawnOut := stdout.String()
	if !strings.Contains(spawnOut, "Session: SESS-") {
		t.Fatalf("stdout = %q, want session id", spawnOut)
	}
	sessionID := extractSessionID(spawnOut)
	if sessionID == "" {
		t.Fatalf("could not extract session id from %q", spawnOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Tmux pane: %5") {
		t.Fatalf("stdout = %q, want pane detail", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"capture", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("capture failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "hello from pane") {
		t.Fatalf("stdout = %q, want captured pane output", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"attach", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Attaching to "+sessionID+" (%5)") {
		t.Fatalf("stdout = %q, want attach summary", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Sessions:") {
		t.Fatalf("stdout = %q, want Sessions section", out)
	}
	if !strings.Contains(out, "agent=backend-main") {
		t.Fatalf("stdout = %q, want session summary row", out)
	}
	if !strings.Contains(out, "Sessions: 1") && !strings.Contains(out, "  Sessions: 1") {
		t.Fatalf("stdout = %q, want session count", out)
	}
}

func TestExecuteTaskCreateShowAndStepList(t *testing.T) {
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "create", "Implement milestone 3", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}

	createOut := stdout.String()
	if !strings.Contains(createOut, "Task created") {
		t.Fatalf("stdout = %q, want Task created", createOut)
	}
	taskID := extractEntityID(createOut, "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", createOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	showOut := stdout.String()
	if !strings.Contains(showOut, "Mode: Direct") {
		t.Fatalf("stdout = %q, want Direct mode", showOut)
	}
	if !strings.Contains(showOut, "Status: Planned") {
		t.Fatalf("stdout = %q, want Planned status", showOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"step", "list", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("step list failed: %v", err)
	}
	stepOut := stdout.String()
	if !strings.Contains(stepOut, "type=implementation") {
		t.Fatalf("stdout = %q, want implementation step", stepOut)
	}
	if !strings.Contains(stepOut, "status=Proposed") {
		t.Fatalf("stdout = %q, want Proposed step", stepOut)
	}
	if !strings.Contains(stepOut, "agent=backend-main") {
		t.Fatalf("stdout = %q, want preferred agent", stepOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Tasks: 1") && !strings.Contains(out, "  Tasks: 1") {
		t.Fatalf("stdout = %q, want task count", out)
	}
	if !strings.Contains(out, "title=Implement milestone 3") {
		t.Fatalf("stdout = %q, want task detail row", out)
	}
	if !strings.Contains(out, "next=confirm the proposed step and move the task to Ready") {
		t.Fatalf("stdout = %q, want recommended next action", out)
	}
	if !strings.Contains(out, "* STEP-") || !strings.Contains(out, "status=Proposed") {
		t.Fatalf("stdout = %q, want task step summary", out)
	}

	artifactDir := filepath.Join(repoRoot, ".aom", "tasks", taskID)
	for _, name := range []string{"task.md", "state.md", "index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join(artifactDir, name)); err != nil {
			t.Fatalf("artifact %s missing: %v", name, err)
		}
	}
}

func TestExecuteTaskUpdateCloseAndStepUpdate(t *testing.T) {
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	if err := Execute([]string{"task", "create", "Implement milestone 3", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")

	stdout.Reset()
	if err := Execute([]string{"step", "list", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("step list failed: %v", err)
	}
	stepID := extractStepID(stdout.String())
	if stepID == "" {
		t.Fatalf("could not extract step id from %q", stdout.String())
	}

	stdout.Reset()
	if err := Execute([]string{"task", "update", taskID, "--mode", "bugfix", "--status", "ready"}, &stdout, &stderr); err != nil {
		t.Fatalf("task update failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Mode: Bugfix") || !strings.Contains(out, "Status: Ready") {
		t.Fatalf("stdout = %q, want updated task fields", out)
	}

	stdout.Reset()
	if err := Execute([]string{"step", "update", stepID, "--status", "confirmed"}, &stdout, &stderr); err != nil {
		t.Fatalf("step update to confirmed failed: %v", err)
	}

	stdout.Reset()
	if err := Execute([]string{"step", "update", stepID, "--status", "ready"}, &stdout, &stderr); err != nil {
		t.Fatalf("step update to ready failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Status: Ready") {
		t.Fatalf("stdout = %q, want ready step", out)
	}

	stdout.Reset()
	if err := Execute([]string{"task", "update", taskID, "--status", "in-progress"}, &stdout, &stderr); err != nil {
		t.Fatalf("task update to in-progress failed: %v", err)
	}

	stdout.Reset()
	if err := Execute([]string{"task", "close", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task close failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Status: Done") {
		t.Fatalf("stdout = %q, want Done status", out)
	}

	stdout.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "status=Done") {
		t.Fatalf("stdout = %q, want done task summary", out)
	}
	if !strings.Contains(out, "next=task is closed; archive later if needed") {
		t.Fatalf("stdout = %q, want closed task next action", out)
	}
	if !strings.Contains(out, "status=Ready") {
		t.Fatalf("stdout = %q, want ready step summary", out)
	}
}

func TestExecuteStatusHighlightsNeedsAttention(t *testing.T) {
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	if err := Execute([]string{"task", "create", "Investigate failing provider", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")

	stdout.Reset()
	if err := Execute([]string{"step", "list", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("step list failed: %v", err)
	}
	stepID := extractStepID(stdout.String())
	if stepID == "" {
		t.Fatalf("could not extract step id from %q", stdout.String())
	}

	stdout.Reset()
	if err := Execute([]string{"task", "update", taskID, "--status", "ready"}, &stdout, &stderr); err != nil {
		t.Fatalf("task update to ready failed: %v", err)
	}
	stdout.Reset()
	if err := Execute([]string{"task", "update", taskID, "--status", "in-progress"}, &stdout, &stderr); err != nil {
		t.Fatalf("task update to in-progress failed: %v", err)
	}
	stdout.Reset()
	if err := Execute([]string{"step", "update", stepID, "--status", "confirmed"}, &stdout, &stderr); err != nil {
		t.Fatalf("step update to confirmed failed: %v", err)
	}
	stdout.Reset()
	if err := Execute([]string{"step", "update", stepID, "--status", "ready"}, &stdout, &stderr); err != nil {
		t.Fatalf("step update to ready failed: %v", err)
	}
	stdout.Reset()
	if err := Execute([]string{"step", "update", stepID, "--status", "in-progress"}, &stdout, &stderr); err != nil {
		t.Fatalf("step update to in-progress failed: %v", err)
	}
	stdout.Reset()
	if err := Execute([]string{"step", "update", stepID, "--status", "needs-attention"}, &stdout, &stderr); err != nil {
		t.Fatalf("step update to needs-attention failed: %v", err)
	}
	stdout.Reset()
	if err := Execute([]string{"task", "update", taskID, "--status", "needs-attention"}, &stdout, &stderr); err != nil {
		t.Fatalf("task update to needs-attention failed: %v", err)
	}

	stdout.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "status=NeedsAttention") {
		t.Fatalf("stdout = %q, want needs-attention status", out)
	}
	if !strings.Contains(out, "next=operator review is needed before work continues") {
		t.Fatalf("stdout = %q, want operator review hint", out)
	}
}

func TestExecuteOpenFailsClearlyWhenTmuxIsUnavailable(t *testing.T) {
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "", errors.New("not found") },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	err = Execute([]string{"open"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("open should fail when tmux is unavailable")
	}
	if !strings.Contains(err.Error(), "ensure tmux workspace: tmux is not available in the current environment") {
		t.Fatalf("error = %q, want missing tmux message", err)
	}
}

func stubAppFactory(t *testing.T, manager *tmux.Manager) func() {
	t.Helper()

	original := newApp
	newApp = func() *app.App {
		return &app.App{
			Planner:  app.New().Planner,
			Projects: project.NewService(),
			Tmux:     manager,
		}
	}

	return func() {
		newApp = original
	}
}

func extractSessionID(output string) string {
	return extractEntityID(output, "Session: ")
}

func extractEntityID(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, prefix))
	}

	return ""
}

func extractStepID(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "- "), " | ", 2)
		if len(parts) == 0 {
			continue
		}
		return strings.TrimSpace(parts[0])
	}

	return ""
}

func TestExecutePlanShowsRecommendation(t *testing.T) {
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"plan", "fix login bug"}, &stdout, &stderr); err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Plan") {
		t.Fatalf("stdout = %q, want Plan header", out)
	}
	if !strings.Contains(out, "Mode: Bugfix") {
		t.Fatalf("stdout = %q, want Bugfix mode", out)
	}
	if !strings.Contains(out, "Recommended agent: backend-main") {
		t.Fatalf("stdout = %q, want backend-main recommendation", out)
	}
	if !strings.Contains(out, "Proposed steps:") {
		t.Fatalf("stdout = %q, want proposed steps", out)
	}
}

func TestExecutePlanCreatePersistsTaskAndSteps(t *testing.T) {
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

	restoreAppFactory := stubAppFactory(t, tmux.NewManagerWithDeps(
		func(string) (string, error) { return "/usr/bin/tmux", nil },
		func(string, ...string) ([]byte, error) { return nil, nil },
		func(string, ...string) error { return nil },
	))
	defer restoreAppFactory()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"project", "init", "my-app", "--repo", repoRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("project init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"plan", "fix login bug", "--create"}, &stdout, &stderr); err != nil {
		t.Fatalf("plan --create failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Task created from plan") {
		t.Fatalf("stdout = %q, want created-from-plan summary", out)
	}
	taskID := extractEntityID(out, "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", out)
	}

	stdout.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	showOut := stdout.String()
	if !strings.Contains(showOut, "Mode: Bugfix") {
		t.Fatalf("stdout = %q, want Bugfix mode", showOut)
	}
	if !strings.Contains(showOut, "Preferred agent: backend-main") {
		t.Fatalf("stdout = %q, want backend-main ownership", showOut)
	}

	stdout.Reset()
	if err := Execute([]string{"step", "list", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("step list failed: %v", err)
	}
	stepOut := stdout.String()
	if !strings.Contains(stepOut, "type=research") {
		t.Fatalf("stdout = %q, want research step", stepOut)
	}
	if !strings.Contains(stepOut, "type=implementation") {
		t.Fatalf("stdout = %q, want implementation step", stepOut)
	}
	if !strings.Contains(stepOut, "dependencies=STEP-") {
		t.Fatalf("stdout = %q, want sequential dependency", stepOut)
	}

	artifactDir := filepath.Join(repoRoot, ".aom", "tasks", taskID)
	if _, err := os.Stat(filepath.Join(artifactDir, "log.md")); err != nil {
		t.Fatalf("plan artifact log missing: %v", err)
	}
}
