package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteVersionPrintsBuildInfo(t *testing.T) {
	oldVersion, oldCommit, oldBuiltAt, oldGoVersion, oldDirty := Version, Commit, BuiltAt, GoVersion, Dirty
	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		BuiltAt = oldBuiltAt
		GoVersion = oldGoVersion
		Dirty = oldDirty
	})

	Version = "v0.3.0"
	Commit = "abc1234"
	BuiltAt = "2026-05-22T10:15:00Z"
	GoVersion = "go1.24.4"
	Dirty = "false"

	var stdout bytes.Buffer
	runner := Runner{stdout: &stdout}

	if err := runner.executeVersion(); err != nil {
		t.Fatalf("executeVersion failed: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{
		"aom version v0.3.0",
		"commit abc1234",
		"built 2026-05-22T10:15:00Z",
		"go go1.24.4",
		"dirty false",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output = %q, want %q", got, want)
		}
	}
}

func TestExecuteSupportsVersionFlag(t *testing.T) {
	oldVersion, oldCommit, oldBuiltAt, oldGoVersion, oldDirty := Version, Commit, BuiltAt, GoVersion, Dirty
	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		BuiltAt = oldBuiltAt
		GoVersion = oldGoVersion
		Dirty = oldDirty
	})

	Version = "v0.3.0"
	Commit = "abc1234"
	BuiltAt = "2026-05-22T10:15:00Z"
	GoVersion = "go1.24.4"
	Dirty = "false"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("Execute(--version) failed: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "aom version v0.3.0") {
		t.Fatalf("stdout = %q, want version output", got)
	}
}
