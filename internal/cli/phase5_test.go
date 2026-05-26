package cli

import (
	"strings"
	"testing"
)

// runCLI is a thin helper that invokes Execute and returns the error.
func runCLI(t *testing.T, args ...string) error {
	t.Helper()
	var buf strings.Builder
	return Execute(args, &buf, &buf)
}

// ── aom task accept --auto flag validation ────────────────────────────────────

func TestTaskAcceptAutoFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing task id",
			args:    []string{"task", "accept", "--auto"},
			wantErr: "task identifier is required",
		},
		{
			name:    "bad --interval value",
			args:    []string{"task", "accept", "--auto", "--interval", "notaduration", "TASK-1"},
			wantErr: "--interval",
		},
		{
			name:    "bad --timeout value",
			args:    []string{"task", "accept", "--auto", "--timeout", "notaduration", "TASK-1"},
			wantErr: "--timeout",
		},
		{
			name:    "--interval missing value",
			args:    []string{"task", "accept", "--auto", "--interval"},
			wantErr: "--interval requires a value",
		},
		{
			name:    "--timeout missing value",
			args:    []string{"task", "accept", "--auto", "--timeout"},
			wantErr: "--timeout requires a value",
		},
		{
			name:    "unknown flag",
			args:    []string{"task", "accept", "--bogus", "TASK-1"},
			wantErr: "unknown flag",
		},
		{
			name:    "two positional args",
			args:    []string{"task", "accept", "TASK-1", "TASK-2"},
			wantErr: "task accept takes exactly one task identifier",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCLI(t, tc.args...)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// ── aom session watch flag validation ────────────────────────────────────────

func TestSessionWatchFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "--auto-spawn requires launch mode",
			args:    []string{"session", "watch", "--auto-spawn"},
			wantErr: "--auto-spawn requires --mock or --real",
		},
		{
			name:    "bad --interval",
			args:    []string{"session", "watch", "--interval", "bad"},
			wantErr: "--interval",
		},
		{
			name:    "bad --timeout",
			args:    []string{"session", "watch", "--timeout", "bad"},
			wantErr: "--timeout",
		},
		{
			name:    "--interval missing value",
			args:    []string{"session", "watch", "--interval"},
			wantErr: "--interval requires a value",
		},
		{
			name:    "unknown flag",
			args:    []string{"session", "watch", "--bogus"},
			wantErr: "unknown flag",
		},
		{
			name:    "--mock and --real conflict",
			args:    []string{"session", "watch", "--mock", "--real"},
			wantErr: "cannot be used together",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCLI(t, tc.args...)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// ── aom run-pipeline flag validation ─────────────────────────────────────────

func TestRunPipelineFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "no args",
			args:    []string{"run-pipeline"},
			wantErr: "task ID is required",
		},
		{
			name:    "no launch mode",
			args:    []string{"run-pipeline", "TASK-1"},
			wantErr: "--mock or --real is required",
		},
		{
			name:    "bad --timeout",
			args:    []string{"run-pipeline", "TASK-1", "--mock", "--timeout", "bad"},
			wantErr: "--timeout",
		},
		{
			name:    "--agent missing value",
			args:    []string{"run-pipeline", "TASK-1", "--mock", "--agent"},
			wantErr: "--agent requires a value",
		},
		{
			name:    "unknown flag",
			args:    []string{"run-pipeline", "TASK-1", "--bogus"},
			wantErr: "unknown flag",
		},
		{
			name:    "--mock and --real conflict",
			args:    []string{"run-pipeline", "TASK-1", "--mock", "--real"},
			wantErr: "cannot be used together",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCLI(t, tc.args...)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// ── parseSpawnItemCommand ─────────────────────────────────────────────────────

func TestParseSpawnItemCommand(t *testing.T) {
	cases := []struct {
		cmd        string
		wantAgent  string
		wantTaskID string
	}{
		{
			cmd:        "aom session spawn backend-main --task TASK-abc-1 --real",
			wantAgent:  "backend-main",
			wantTaskID: "TASK-abc-1",
		},
		{
			cmd:        "aom session spawn frontend-main --task TASK-xyz-2 --mock",
			wantAgent:  "frontend-main",
			wantTaskID: "TASK-xyz-2",
		},
		{
			cmd:       "aom session spawn reviewer-main --real",
			wantAgent: "reviewer-main",
		},
		{
			cmd: "not a spawn command",
		},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			gotAgent, gotTask := parseSpawnItemCommand(tc.cmd)
			if gotAgent != tc.wantAgent {
				t.Errorf("agent: got %q, want %q", gotAgent, tc.wantAgent)
			}
			if gotTask != tc.wantTaskID {
				t.Errorf("taskID: got %q, want %q", gotTask, tc.wantTaskID)
			}
		})
	}
}
