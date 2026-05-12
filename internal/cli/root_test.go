package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
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
			case "display-message":
				return []byte("%5\n"), nil
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
			case "display-message":
				return []byte("%5\n"), nil
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
	if !strings.Contains(out, "Worktree status: Planned") {
		t.Fatalf("stdout = %q, want planned worktree status for non-git repo", out)
	}
	spawnWorktreePath := extractLineValue(out, "Worktree path: ")
	if spawnWorktreePath == "" {
		t.Fatalf("stdout = %q, want worktree path", out)
	}
	if same, err := samePath(spawnWorktreePath, repoRoot); err != nil {
		t.Fatalf("samePath failed: %v", err)
	} else if !same {
		t.Fatalf("worktree path = %q, want repo root %q", spawnWorktreePath, repoRoot)
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
	} else if worktreePath := extractLineValue(out, "Worktree: "); func() bool {
		same, err := samePath(worktreePath, repoRoot)
		if err != nil {
			t.Fatalf("samePath failed: %v", err)
		}
		return same
	}() == false {
		t.Fatalf("worktree path = %q, want repo root %q", worktreePath, repoRoot)
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
	if !strings.Contains(string(logData), "session.ready") {
		t.Fatalf("log.md = %q, want session.ready event", string(logData))
	}
	if !strings.Contains(string(logData), "Session Booting") {
		t.Fatalf("log.md = %q, want booting lifecycle state", string(logData))
	}
	if !strings.Contains(string(logData), "Session Idle") {
		t.Fatalf("log.md = %q, want idle lifecycle state", string(logData))
	}
}

func TestExecuteSessionSpawnWithTaskLogsFailureWhenPaneCreationFails(t *testing.T) {
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
				return nil, errors.New("split failed")
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
	if err := Execute([]string{"task", "create", "Bind failing session to task", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Execute([]string{"session", "spawn", "backend-main", "--task", taskID, "--mock"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("session spawn should fail when pane creation fails")
	}
	if !strings.Contains(err.Error(), "split failed") {
		t.Fatalf("error = %q, want split failure", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "list"}, &stdout, &stderr); err != nil {
		t.Fatalf("session list failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "status=Failed") {
		t.Fatalf("stdout = %q, want failed session status", out)
	}

	logData, err := os.ReadFile(filepath.Join(repoRoot, ".aom", "tasks", taskID, "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "session.failed") {
		t.Fatalf("log.md = %q, want session.failed event", string(logData))
	}
	if !strings.Contains(string(logData), "session.created") {
		t.Fatalf("log.md = %q, want session.created event before failure", string(logData))
	}
	if !strings.Contains(string(logData), "Session Failed") {
		t.Fatalf("log.md = %q, want failed lifecycle state", string(logData))
	}
}

func TestExecuteSessionSpawnWithTaskLogsFailureWhenPaneAnnotationFails(t *testing.T) {
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
				return nil, errors.New("annotate failed")
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
	if err := Execute([]string{"task", "create", "Bind annotated session to task", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Execute([]string{"session", "spawn", "backend-main", "--task", taskID, "--mock"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("session spawn should fail when pane annotation fails")
	}
	if !strings.Contains(err.Error(), "annotate failed") {
		t.Fatalf("error = %q, want annotate failure", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "list"}, &stdout, &stderr); err != nil {
		t.Fatalf("session list failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "status=Failed") {
		t.Fatalf("stdout = %q, want failed session status", out)
	}

	logData, err := os.ReadFile(filepath.Join(repoRoot, ".aom", "tasks", taskID, "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "session.created") {
		t.Fatalf("log.md = %q, want session.created event", string(logData))
	}
	if !strings.Contains(string(logData), "session.ready") {
		t.Fatalf("log.md = %q, want session.ready event before annotation failure", string(logData))
	}
	if !strings.Contains(string(logData), "session.failed") {
		t.Fatalf("log.md = %q, want session.failed event", string(logData))
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

func TestExecuteAttachLogsOperatorInterventionForTaskBoundSession(t *testing.T) {
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
	if err := Execute([]string{"task", "create", "Intervene in active task", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
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
	sessionID := extractSessionID(stdout.String())
	if sessionID == "" {
		t.Fatalf("could not extract session id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"attach", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Attaching to "+sessionID+" (%5)") {
		t.Fatalf("stdout = %q, want attach summary", out)
	}

	indexData, err := os.ReadFile(filepath.Join(repoRoot, ".aom", "tasks", taskID, "index.md"))
	if err != nil {
		t.Fatalf("ReadFile(index.md) failed: %v", err)
	}
	if !strings.Contains(string(indexData), "Active Session: "+sessionID) {
		t.Fatalf("index.md = %q, want active session after attach", string(indexData))
	}

	logData, err := os.ReadFile(filepath.Join(repoRoot, ".aom", "tasks", taskID, "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "operator.intervention") {
		t.Fatalf("log.md = %q, want operator.intervention event", string(logData))
	}
	if !strings.Contains(string(logData), "Re-analysis required") {
		t.Fatalf("log.md = %q, want re-analysis marker", string(logData))
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
	if !strings.Contains(showOut, "Worktree status: Planned") {
		t.Fatalf("stdout = %q, want planned worktree status", showOut)
	}
	if !strings.Contains(showOut, "Worktree branch: aom/") {
		t.Fatalf("stdout = %q, want worktree branch", showOut)
	}
	if !strings.Contains(showOut, ".aom") || !strings.Contains(showOut, "worktrees") {
		t.Fatalf("stdout = %q, want worktree path", showOut)
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
	if !strings.Contains(out, "worktree=Planned | branch=aom/") {
		t.Fatalf("stdout = %q, want planned worktree summary", out)
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

func TestExecuteTaskCreateProvisionsWorktreeWhenRepoIsGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for worktree provisioning integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

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
	if err := Execute([]string{"task", "create", "Implement worktree provisioning", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	showOut := stdout.String()
	if !strings.Contains(showOut, "Worktree status: Ready") {
		t.Fatalf("stdout = %q, want ready worktree status", showOut)
	}
	if !strings.Contains(showOut, ".aom") || !strings.Contains(showOut, "worktrees") {
		t.Fatalf("stdout = %q, want worktree path", showOut)
	}
	worktreePath := extractLineValue(showOut, "Worktree path: ")
	if worktreePath == "" {
		t.Fatalf("could not extract worktree path from %q", showOut)
	}
	for _, name := range []string{"task.md", "state.md", "index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join(worktreePath, ".agent", name)); err != nil {
			t.Fatalf("artifact %s missing in worktree .agent: %v", name, err)
		}
	}
}

func TestExecuteSessionSpawnUsesProvisionedWorktreeWhenRepoIsGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for worktree provisioning integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

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
			case "display-message":
				return []byte("%5\n"), nil
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
	if err := Execute([]string{"task", "create", "Spawn in provisioned worktree", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
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
	spawnOut := stdout.String()
	if !strings.Contains(spawnOut, "Worktree status: Active") {
		t.Fatalf("stdout = %q, want active worktree status", spawnOut)
	}
	if !strings.Contains(spawnOut, ".aom") || !strings.Contains(spawnOut, "worktrees") {
		t.Fatalf("stdout = %q, want provisioned worktree path", spawnOut)
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
	showOut := stdout.String()
	if !strings.Contains(showOut, "Task: "+taskID) {
		t.Fatalf("stdout = %q, want task in session show", showOut)
	}
	if !strings.Contains(showOut, ".aom") || !strings.Contains(showOut, "worktrees") {
		t.Fatalf("stdout = %q, want provisioned worktree path", showOut)
	}
	worktreePath := extractLineValue(showOut, "Worktree: ")
	if worktreePath == "" {
		t.Fatalf("could not extract worktree path from %q", showOut)
	}
	indexData, err := os.ReadFile(filepath.Join(worktreePath, ".agent", "index.md"))
	if err != nil {
		t.Fatalf("ReadFile(index.md) failed: %v", err)
	}
	if !strings.Contains(string(indexData), "Active Session: "+sessionID) {
		t.Fatalf("index.md = %q, want active session in worktree artifact", string(indexData))
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "worktree=Active | branch=aom/") {
		t.Fatalf("stdout = %q, want active worktree summary", out)
	}
}

func TestExecuteStatusMarksStaleWorktreeNeedsRepair(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for worktree repair integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

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
	if err := Execute([]string{"task", "create", "Repair stale worktree", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	worktreePath := extractLineValue(stdout.String(), "Worktree path: ")
	if worktreePath == "" {
		t.Fatalf("could not extract worktree path from %q", stdout.String())
	}

	if err := os.RemoveAll(worktreePath); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	statusOut := stdout.String()
	if !strings.Contains(statusOut, "worktree=NeedsRepair | branch=aom/") {
		t.Fatalf("stdout = %q, want worktree needs-repair summary", statusOut)
	}
	if !strings.Contains(statusOut, "repair=run \"aom worktree repair "+taskID+"\" or inspect the git worktree path before continuing") {
		t.Fatalf("stdout = %q, want repair hint", statusOut)
	}
	if !strings.Contains(statusOut, "next=repair the task worktree before continuing") {
		t.Fatalf("stdout = %q, want repair next action", statusOut)
	}
}

func TestExecuteWorktreeRepairRestoresMissingGitWorktreeAndArtifacts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for worktree repair integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

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
	if err := Execute([]string{"task", "create", "Repair command smoke", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	taskID := extractEntityID(stdout.String(), "Task: ")
	if taskID == "" {
		t.Fatalf("could not extract task id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	worktreePath := extractLineValue(stdout.String(), "Worktree path: ")
	if worktreePath == "" {
		t.Fatalf("could not extract worktree path from %q", stdout.String())
	}

	if err := os.RemoveAll(worktreePath); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"worktree", "repair", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("worktree repair failed: %v", err)
	}
	repairOut := stdout.String()
	if !strings.Contains(repairOut, "Worktree repaired") {
		t.Fatalf("stdout = %q, want repair confirmation", repairOut)
	}
	if !strings.Contains(repairOut, "Status: Ready") {
		t.Fatalf("stdout = %q, want ready status after repair", repairOut)
	}

	for _, name := range []string{"task.md", "state.md", "index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join(worktreePath, ".agent", name)); err != nil {
			t.Fatalf("artifact %s missing after repair: %v", name, err)
		}
	}

	logData, err := os.ReadFile(filepath.Join(worktreePath, ".agent", "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "worktree.repaired") {
		t.Fatalf("log.md = %q, want worktree.repaired event", string(logData))
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	showOut := stdout.String()
	if !strings.Contains(showOut, "Worktree status: Ready") {
		t.Fatalf("stdout = %q, want ready worktree status", showOut)
	}
}

func TestExecuteStatusReconcilesDetachedSessionAndDowngradesWorktreeToReady(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for session/worktree reconciliation integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

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
			case "display-message":
				return nil, errors.New("pane not found")
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
	if err := Execute([]string{"task", "create", "Reconcile missing pane", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
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
	sessionID := extractSessionID(stdout.String())
	if sessionID == "" {
		t.Fatalf("could not extract session id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	statusOut := stdout.String()
	if !strings.Contains(statusOut, "status=Detached") {
		t.Fatalf("stdout = %q, want detached session status", statusOut)
	}
	if !strings.Contains(statusOut, "worktree=Ready | branch=aom/") {
		t.Fatalf("stdout = %q, want worktree downgraded to Ready", statusOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Status: Detached") {
		t.Fatalf("stdout = %q, want detached status in session show", out)
	}
}

func TestExecuteSessionStopMarksStoppedAndDowngradesWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for session stop integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

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
			case "display-message":
				return []byte("%5\n"), nil
			case "kill-pane":
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
	if err := Execute([]string{"task", "create", "Stop live session", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
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
	sessionID := extractSessionID(stdout.String())
	if sessionID == "" {
		t.Fatalf("could not extract session id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "stop", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session stop failed: %v", err)
	}
	stopOut := stdout.String()
	if !strings.Contains(stopOut, "Session stopped") {
		t.Fatalf("stdout = %q, want stop confirmation", stopOut)
	}
	if !strings.Contains(stopOut, "Status: Stopped") {
		t.Fatalf("stdout = %q, want stopped status", stopOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Status: Stopped") {
		t.Fatalf("stdout = %q, want stopped status in session show", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	statusOut := stdout.String()
	if !strings.Contains(statusOut, "status=Stopped") {
		t.Fatalf("stdout = %q, want stopped session in status", statusOut)
	}
	if !strings.Contains(statusOut, "worktree=Ready | branch=aom/") {
		t.Fatalf("stdout = %q, want worktree downgraded to Ready", statusOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	worktreePath := extractLineValue(stdout.String(), "Worktree path: ")
	if worktreePath == "" {
		t.Fatalf("could not extract worktree path from %q", stdout.String())
	}
	logData, err := os.ReadFile(filepath.Join(worktreePath, ".agent", "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "session.stopped") {
		t.Fatalf("log.md = %q, want session.stopped event", string(logData))
	}
}

func TestExecuteSessionArchiveMarksStoppedSessionArchived(t *testing.T) {
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
		func(name string, args ...string) ([]byte, error) {
			if len(args) == 0 {
				return nil, nil
			}
			switch args[0] {
			case "has-session":
				return nil, errors.New("session not found")
			case "new-session":
				return nil, nil
			case "split-window":
				return []byte("@1 %5\n"), nil
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
	sessionID := extractSessionID(stdout.String())
	if sessionID == "" {
		t.Fatalf("could not extract session id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "stop", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session stop failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "archive", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session archive failed: %v", err)
	}
	archiveOut := stdout.String()
	if !strings.Contains(archiveOut, "Session archived") {
		t.Fatalf("stdout = %q, want archive confirmation", archiveOut)
	}
	if !strings.Contains(archiveOut, "Status: Archived") {
		t.Fatalf("stdout = %q, want archived status", archiveOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", sessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Status: Archived") {
		t.Fatalf("stdout = %q, want archived status in session show", out)
	}
}

func TestExecuteSessionReplaceSupersedesOldSessionInSameTaskWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for session replace integration test")
	}

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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")

	firstHasSession := true
	splitCount := 0
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
				splitCount++
				if splitCount == 1 {
					return []byte("@1 %5\n"), nil
				}
				return []byte("@1 %6\n"), nil
			case "set-option":
				return nil, nil
			case "display-message":
				target := args[len(args)-2]
				if target == "%5" {
					return []byte("%5\n"), nil
				}
				if target == "%6" {
					return []byte("%6\n"), nil
				}
				return nil, errors.New("pane not found")
			case "kill-pane":
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
	if err := Execute([]string{"task", "create", "Replace live session", "--role", "backend", "--agent", "backend-main"}, &stdout, &stderr); err != nil {
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
	oldSessionID := extractSessionID(stdout.String())
	if oldSessionID == "" {
		t.Fatalf("could not extract old session id from %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "replace", oldSessionID, "--agent", "reviewer-main", "--reason", "provider limit", "--mock"}, &stdout, &stderr); err != nil {
		t.Fatalf("session replace failed: %v", err)
	}
	replaceOut := stdout.String()
	if !strings.Contains(replaceOut, "Session replaced") {
		t.Fatalf("stdout = %q, want replace confirmation", replaceOut)
	}
	newSessionID := extractEntityID(replaceOut, "New session: ")
	if newSessionID == "" {
		t.Fatalf("could not extract new session id from %q", replaceOut)
	}
	if newSessionID == oldSessionID {
		t.Fatalf("new session id = old session id = %q", newSessionID)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", oldSessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("old session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Status: Stopped") {
		t.Fatalf("stdout = %q, want stopped old session", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"session", "show", newSessionID}, &stdout, &stderr); err != nil {
		t.Fatalf("new session show failed: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "Agent: reviewer-main") || !strings.Contains(out, "Task: "+taskID) {
		t.Fatalf("stdout = %q, want reviewer replacement bound to same task", out)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"status"}, &stdout, &stderr); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	statusOut := stdout.String()
	if !strings.Contains(statusOut, "status=Stopped") {
		t.Fatalf("stdout = %q, want stopped old session in summary", statusOut)
	}
	if !strings.Contains(statusOut, newSessionID) {
		t.Fatalf("stdout = %q, want replacement session in summary", statusOut)
	}
	if !strings.Contains(statusOut, "worktree=Active | branch=aom/") {
		t.Fatalf("stdout = %q, want active worktree retained by replacement session", statusOut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"task", "show", taskID}, &stdout, &stderr); err != nil {
		t.Fatalf("task show failed: %v", err)
	}
	worktreePath := extractLineValue(stdout.String(), "Worktree path: ")
	if worktreePath == "" {
		t.Fatalf("could not extract worktree path from %q", stdout.String())
	}
	logData, err := os.ReadFile(filepath.Join(worktreePath, ".agent", "log.md"))
	if err != nil {
		t.Fatalf("ReadFile(log.md) failed: %v", err)
	}
	if !strings.Contains(string(logData), "session.replaced") {
		t.Fatalf("log.md = %q, want session.replaced event", string(logData))
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

func extractLineValue(output, prefix string) string {
	return extractEntityID(output, prefix)
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
	if !strings.Contains(showOut, "Worktree status: Planned") {
		t.Fatalf("stdout = %q, want planned worktree status", showOut)
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

func samePath(left, right string) (bool, error) {
	leftEval, err := filepath.EvalSymlinks(left)
	if err != nil {
		return false, err
	}
	rightEval, err := filepath.EvalSymlinks(right)
	if err != nil {
		return false, err
	}

	return filepath.Clean(leftEval) == filepath.Clean(rightEval), nil
}
