package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var systemInstallPath = "/usr/local/bin/aom"

func (r Runner) executeUninstall() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	candidates := []string{
		systemInstallPath,
		filepath.Join(homeDir, ".local", "bin", "aom"),
	}

	removed := 0
	for _, path := range candidates {
		if _, statErr := os.Lstat(path); statErr != nil {
			continue
		}
		if err := removePath(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		fmt.Fprintf(r.stdout, "removed %s\n", path)
		removed++
	}

	if removed == 0 {
		fmt.Fprintln(r.stdout, "No installed aom binary found in /usr/local/bin or ~/.local/bin.")
	} else {
		fmt.Fprintln(r.stdout, "Uninstall complete.")
	}

	return nil
}

func uninstallHelpLine() string {
	return fmt.Sprintf("aom uninstall : remove installed binaries from %s and ~/.local/bin/aom", systemInstallPath)
}

func removePath(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}

	if path != systemInstallPath {
		return os.Remove(path)
	}

	if _, lookErr := exec.LookPath("sudo"); lookErr != nil {
		return os.Remove(path)
	}

	cmd := exec.Command("sudo", "rm", "-f", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
