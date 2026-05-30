package cli

import (
	"strings"
	"testing"
)

// ── aom team view flag validation ─────────────────────────────────────────────

func TestTeamViewFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "--layout missing value",
			args:    []string{"team", "view", "--layout"},
			wantErr: "--layout requires a value",
		},
		{
			name:    "unknown flag",
			args:    []string{"team", "view", "--bogus"},
			wantErr: "unknown flag",
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

// ── aom orchestrate flag validation ──────────────────────────────────────────

func TestOrchestrateFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "--layout missing value",
			args:    []string{"orchestrate", "--layout"},
			wantErr: "--layout requires a value",
		},
		{
			name:    "unknown flag",
			args:    []string{"orchestrate", "--bogus"},
			wantErr: "unknown flag",
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

// ── aom session spawn --grid flag validation ──────────────────────────────────

func TestSessionSpawnGridFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "--layout missing value",
			args:    []string{"session", "spawn", "some-agent", "--layout"},
			wantErr: "--layout requires a value",
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

// ── aom team subcommand dispatch ──────────────────────────────────────────────

func TestTeamUnknownSubcommand(t *testing.T) {
	err := runCLI(t, "team", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown team subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "unknown team command") {
		t.Fatalf("error %q does not mention 'unknown team command'", err.Error())
	}
}

func TestTeamNoSubcommand(t *testing.T) {
	err := runCLI(t, "team")
	if err == nil {
		t.Fatal("expected error for missing team subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "subcommand is required") {
		t.Fatalf("error %q does not mention 'subcommand is required'", err.Error())
	}
}

// ── aom orchestrate no-project error path ────────────────────────────────────

func TestOrchestrateNoProject(t *testing.T) {
	// Running orchestrate outside a project directory should fail gracefully.
	err := runCLI(t, "orchestrate")
	if err == nil {
		t.Fatal("expected error when no project is present")
	}
	// The error will be from Projects.Open — not a flag parse error.
	// We just verify it does not panic.
}
