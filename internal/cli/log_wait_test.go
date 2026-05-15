package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTailMultiTaskLogEventsStreamsNewLines(t *testing.T) {
	dir := t.TempDir()

	logA := filepath.Join(dir, "a.md")
	logB := filepath.Join(dir, "b.md")
	if err := os.WriteFile(logA, []byte("existing line\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logB, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	tasks := []taskLogEntry{
		{TaskID: "TASK-1", LogPath: logA},
		{TaskID: "TASK-2", LogPath: logB},
	}

	// Append new content to both logs after a short delay.
	go func() {
		time.Sleep(100 * time.Millisecond)
		f, _ := os.OpenFile(logA, os.O_APPEND|os.O_WRONLY, 0600)
		f.WriteString("### new event A\n")
		f.Close()

		f, _ = os.OpenFile(logB, os.O_APPEND|os.O_WRONLY, 0600)
		f.WriteString("### new event B\n")
		f.Close()
	}()

	var buf bytes.Buffer
	// 3 seconds allows at least one 2-second poll cycle to complete.
	_ = tailMultiTaskLogEvents(&buf, tasks, 3*time.Second)

	out := buf.String()
	if !strings.Contains(out, "[TASK-1] ### new event A") {
		t.Errorf("output missing TASK-1 event; got:\n%s", out)
	}
	if !strings.Contains(out, "[TASK-2] ### new event B") {
		t.Errorf("output missing TASK-2 event; got:\n%s", out)
	}
	// Pre-existing lines must not appear.
	if strings.Contains(out, "existing line") {
		t.Errorf("output should not contain pre-existing lines; got:\n%s", out)
	}
}

func TestWaitForMultiTaskLogEventMatchesAnyTask(t *testing.T) {
	dir := t.TempDir()

	logA := filepath.Join(dir, "a.md")
	logB := filepath.Join(dir, "b.md")
	if err := os.WriteFile(logA, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logB, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	tasks := []taskLogEntry{
		{TaskID: "TASK-A", LogPath: logA},
		{TaskID: "TASK-B", LogPath: logB},
	}

	// Write the target event to logB after a short delay.
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.WriteFile(logB, []byte("### 2026-01-01 | EVT-1 | task.completed | OK\n"), 0600)
	}()

	taskID, line, err := waitForMultiTaskLogEvent(tasks, "task.completed", 5*time.Second)
	if err != nil {
		t.Fatalf("waitForMultiTaskLogEvent failed: %v", err)
	}
	if taskID != "TASK-B" {
		t.Errorf("taskID = %q, want TASK-B", taskID)
	}
	if !strings.Contains(line, "task.completed") {
		t.Errorf("line = %q, want task.completed", line)
	}
}

func TestWaitForMultiTaskLogEventTimesOut(t *testing.T) {
	dir := t.TempDir()
	logA := filepath.Join(dir, "a.md")
	os.WriteFile(logA, []byte(""), 0600)

	tasks := []taskLogEntry{{TaskID: "TASK-A", LogPath: logA}}

	_, _, err := waitForMultiTaskLogEvent(tasks, "task.completed", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want timeout message", err)
	}
}
