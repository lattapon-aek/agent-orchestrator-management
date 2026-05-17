package config

import (
	"os"
	"testing"
)

// TestMain seeds the known-runtime registry with the built-in provider names
// before any test in this package runs. In production code this registration
// happens via internal/app init(), which the config package cannot import.
func TestMain(m *testing.M) {
	for _, name := range []string{"claude", "codex", "gemini", "kiro"} {
		RegisterKnownRuntime(name)
	}
	os.Exit(m.Run())
}
