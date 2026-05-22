package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteInstallRunsRepoScript(t *testing.T) {
	repoRoot := t.TempDir()
	createScript(t, filepath.Join(repoRoot, "scripts", "install.sh"), "#!/usr/bin/env bash\nprintf 'install-ok %s\\n' \"$*\"\n")
	createScript(t, filepath.Join(repoRoot, "scripts", "update.sh"), "#!/usr/bin/env bash\nprintf 'update-ok %s\\n' \"$*\"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	nested := filepath.Join(repoRoot, "worktrees", "task-1")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"install", "--test"}, &stdout, &stderr); err != nil {
		t.Fatalf("Execute(install) failed: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "install-ok --test") {
		t.Fatalf("stdout = %q, want install script output", got)
	}
}

func TestExecuteUpdateRunsRepoScript(t *testing.T) {
	repoRoot := t.TempDir()
	createScript(t, filepath.Join(repoRoot, "scripts", "install.sh"), "#!/usr/bin/env bash\nprintf 'install-ok %s\\n' \"$*\"\n")
	createScript(t, filepath.Join(repoRoot, "scripts", "update.sh"), "#!/usr/bin/env bash\nprintf 'update-ok %s\\n' \"$*\"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	nested := filepath.Join(repoRoot, "worktrees", "task-2")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"update", "--test"}, &stdout, &stderr); err != nil {
		t.Fatalf("Execute(update) failed: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "update-ok --test") {
		t.Fatalf("stdout = %q, want update script output", got)
	}
}

func createScript(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}
}
