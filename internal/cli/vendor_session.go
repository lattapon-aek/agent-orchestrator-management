package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// claudeSessionForWorktree polls ~/.claude/projects/<path-hash>/ for the newest .jsonl
// session file whose mtime is at or after spawnedAt. Returns the UUID (filename without
// .jsonl extension) on success, or an empty string if none is found within timeout.
func claudeSessionForWorktree(worktreePath string, spawnedAt time.Time, timeout time.Duration) (string, error) {
	projectsDir, err := claudeProjectsDirForPath(worktreePath)
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		var newest string
		var newestTime time.Time
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(spawnedAt) {
				continue
			}
			if newest == "" || info.ModTime().After(newestTime) {
				newest = strings.TrimSuffix(entry.Name(), ".jsonl")
				newestTime = info.ModTime()
			}
		}

		if newest != "" {
			return newest, nil
		}

		time.Sleep(time.Second)
	}

	return "", nil
}

// claudeProjectsDirForPath returns the ~/.claude/projects/ subdirectory that Claude
// uses for the given worktree path. Claude encodes the path by replacing every '/'
// and '.' character with '-'.
func claudeProjectsDirForPath(worktreePath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}

	encoded := strings.NewReplacer("/", "-", ".", "-").Replace(worktreePath)
	return filepath.Join(home, ".claude", "projects", encoded), nil
}
