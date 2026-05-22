package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (r Runner) executeInstall(args []string) error {
	return r.executeRepoScript("scripts/install.sh", args)
}

func (r Runner) executeUpdate(args []string) error {
	return r.executeRepoScript("scripts/update.sh", args)
}

func (r Runner) executeRepoScript(relPath string, args []string) error {
	scriptPath, err := findRepoScript(relPath)
	if err != nil {
		return err
	}

	cmd := exec.Command(scriptPath, args...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	cmd.Stdin = r.stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", scriptPath, err)
	}
	return nil
}

func findRepoScript(relPath string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	dir := wd
	for {
		candidate := filepath.Join(dir, relPath)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("%s not found from %s; run this command inside an AOM repository checkout", relPath, wd)
}
