package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const channelFileHeader = "# AOM Team Channel\n\n## Messages\n\n"

func channelFilePath(repoPath string) string {
	return filepath.Join(repoPath, ".aom", "channel.md")
}

func appendChannelMessage(repoPath, agentName, message string, now time.Time) error {
	path := channelFilePath(repoPath)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create channel dir: %w", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(channelFileHeader), 0o644); err != nil {
			return fmt.Errorf("create channel file: %w", err)
		}
	}

	msgID := "MSG-" + strconv.FormatInt(now.UnixNano(), 10)
	entry := fmt.Sprintf("### %s | %s | %s\n- Summary: %s\n\n",
		now.Format(time.RFC3339),
		msgID,
		agentName,
		message,
	)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open channel file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("write channel entry: %w", err)
	}

	return nil
}

func readChannelFile(repoPath string) (string, error) {
	data, err := os.ReadFile(channelFilePath(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read channel file: %w", err)
	}
	return string(data), nil
}
