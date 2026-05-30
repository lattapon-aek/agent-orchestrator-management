package project

import (
	"strings"
	"testing"
)

// TestRenderOrchestratorProfileConditionals verifies that orchestrator.md.tmpl
// renders provider-specific blocks correctly for claude and codex runtimes.
func TestRenderOrchestratorProfileConditionals(t *testing.T) {
	tests := []struct {
		runtime     string
		mustContain []string
		mustExclude []string
	}{
		{
			runtime: "claude",
			mustContain: []string{
				"Running AOM from Your Session",
				"--dangerously-skip-permissions",
				"aom worktree commit --local",
			},
			mustExclude: []string{
				// codex-specific orchestrator block must not appear for claude
				"Run every command sequentially in the foreground",
			},
		},
		{
			runtime: "codex",
			mustContain: []string{
				"Running AOM from Your Session",
				"never use background terminals",
				"foreground",
				"aom worktree commit --local",
			},
			mustExclude: []string{
				"--dangerously-skip-permissions",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.runtime, func(t *testing.T) {
			profile, err := renderAgentProfile(
				"orchestrator-main",
				"orchestrator",
				tc.runtime,
				"orchestrator",
				"", // no override templateDir
				"", // no override aomPath
			)
			if err != nil {
				t.Fatalf("renderAgentProfile(%q): %v", tc.runtime, err)
			}

			for _, want := range tc.mustContain {
				if !strings.Contains(profile, want) {
					t.Errorf("runtime=%q: profile missing expected string %q", tc.runtime, want)
				}
			}
			for _, banned := range tc.mustExclude {
				if strings.Contains(profile, banned) {
					t.Errorf("runtime=%q: profile should not contain %q", tc.runtime, banned)
				}
			}
		})
	}
}
