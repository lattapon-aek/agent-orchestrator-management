package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteUninstallRemovesInstalledBinaries(t *testing.T) {
	oldSystemInstallPath := systemInstallPath
	t.Cleanup(func() {
		systemInstallPath = oldSystemInstallPath
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	systemInstallPath = filepath.Join(homeDir, "system", "aom")
	localInstallPath := filepath.Join(homeDir, ".local", "bin", "aom")

	if err := os.MkdirAll(filepath.Dir(systemInstallPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(system) failed: %v", err)
	}
	if err := os.WriteFile(systemInstallPath, []byte("system"), 0o755); err != nil {
		t.Fatalf("WriteFile(system) failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(localInstallPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(local) failed: %v", err)
	}
	if err := os.WriteFile(localInstallPath, []byte("local"), 0o755); err != nil {
		t.Fatalf("WriteFile(local) failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"uninstall"}, &stdout, &stderr); err != nil {
		t.Fatalf("Execute(uninstall) failed: %v", err)
	}

	for _, path := range []string{systemInstallPath, localInstallPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("Stat(%q) = %v, want removed", path, err)
		}
	}

	got := stdout.String()
	for _, want := range []string{"removed", "Uninstall complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestExecuteUninstallReportsWhenNothingInstalled(t *testing.T) {
	oldSystemInstallPath := systemInstallPath
	t.Cleanup(func() {
		systemInstallPath = oldSystemInstallPath
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	systemInstallPath = filepath.Join(homeDir, "system", "aom")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"uninstall"}, &stdout, &stderr); err != nil {
		t.Fatalf("Execute(uninstall) failed: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "No installed aom binary found") {
		t.Fatalf("stdout = %q, want no-install message", got)
	}
}
