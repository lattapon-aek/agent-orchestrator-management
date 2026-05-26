package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// executeDashboard runs a live-refreshing terminal dashboard that shows
// all sessions, action items, and recent channel messages in one view.
//
// Usage: aom dashboard [--interval <dur>]
//
// Default refresh interval: 5s.  Press Ctrl+C to exit.
func (r Runner) executeDashboard(args []string) error {
	interval := 5 * time.Second

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--interval":
			i++
			if i >= len(args) {
				return fmt.Errorf("--interval requires a value (e.g. 5s, 10s, 1m)")
			}
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return fmt.Errorf("--interval: %w", err)
			}
			interval = d
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	// Handle Ctrl+C / SIGTERM gracefully.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Draw immediately, then refresh on each tick.
	for {
		if err := r.drawDashboard(interval); err != nil {
			// Restore cursor on error.
			fmt.Fprint(r.stdout, "\033[?25h")
			return err
		}
		select {
		case <-ctx.Done():
			fmt.Fprint(r.stdout, "\033[?25h") // restore cursor
			fmt.Fprintln(r.stdout, "")
			fmt.Fprintln(r.stdout, "Dashboard stopped.")
			return nil
		case <-time.After(interval):
		}
	}
}

// drawDashboard clears the terminal and prints one full dashboard frame.
func (r Runner) drawDashboard(interval time.Duration) error {
	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	sessions, err := r.loadProjectSessions(result)
	if err != nil {
		return err
	}

	taskViews, err := r.loadTaskViews(result, sessions)
	if err != nil {
		return err
	}

	items := r.buildActionItems(result, sessions, taskViews)
	channelLines := dashboardChannelTail(result.Project.RepoPath, 6)

	// ── Clear screen: hide cursor, home, clear ────────────────────────────
	fmt.Fprint(r.stdout, "\033[?25l\033[H\033[2J")

	now := time.Now().Format("2006-01-02 15:04:05")
	header := fmt.Sprintf("AOM Dashboard  |  %s  |  %s  |  every %s  |  Ctrl+C to exit",
		result.Project.Name, now, interval)
	fmt.Fprintln(r.stdout, colorize(header, ansiBold, r.stdout))
	fmt.Fprintln(r.stdout, strings.Repeat("─", 78))

	// ── Sessions ─────────────────────────────────────────────────────────
	fmt.Fprintln(r.stdout, colorize("Sessions", ansiBold, r.stdout))
	if len(sessions) == 0 {
		fmt.Fprintln(r.stdout, "  (none)")
	} else {
		for _, s := range sessions {
			statusCol := colorStatus(s.Status, r.stdout)
			taskCol := s.TaskID
			if taskCol == "" {
				taskCol = "—"
			}
			paneCol := "no pane"
			if s.TmuxPane != "" {
				alive, _ := r.app.Tmux.PaneExists(s.TmuxPane)
				if alive {
					paneCol = colorize("live", ansiGreen, r.stdout)
				} else {
					paneCol = colorize("dead", ansiRed, r.stdout)
				}
			}
			fmt.Fprintf(r.stdout, "  %-16s  %-22s  task=%-18s  pane=%s\n",
				s.AgentName, statusCol, taskCol, paneCol)
		}
	}
	fmt.Fprintln(r.stdout, strings.Repeat("─", 78))

	// ── Action Items ──────────────────────────────────────────────────────
	fmt.Fprintln(r.stdout, colorize("Action Items", ansiBold, r.stdout))
	if len(items) == 0 {
		fmt.Fprintln(r.stdout, "  Nothing needs attention")
	} else {
		for _, item := range items {
			priorityColor := ansiYellow
			if item.priority == 1 {
				priorityColor = ansiRed
			} else if item.priority == 3 {
				priorityColor = ansiDim
			}
			tag := colorize(fmt.Sprintf("[%s]", item.label), priorityColor, r.stdout)
			detail := item.detail
			// Truncate long detail lines to keep the dashboard tidy.
			if len(detail) > 55 {
				detail = detail[:52] + "..."
			}
			fmt.Fprintf(r.stdout, "  %s  %s\n", tag, detail)
			if item.command != "" {
				fmt.Fprintf(r.stdout, "         → %s\n", item.command)
			}
		}
	}
	fmt.Fprintln(r.stdout, strings.Repeat("─", 78))

	// ── Recent Channel ────────────────────────────────────────────────────
	fmt.Fprintln(r.stdout, colorize("Recent Channel", ansiBold, r.stdout))
	if len(channelLines) == 0 {
		fmt.Fprintln(r.stdout, "  (no messages yet)")
	} else {
		for _, line := range channelLines {
			if len(line) > 76 {
				line = line[:73] + "..."
			}
			fmt.Fprintf(r.stdout, "  %s\n", line)
		}
	}
	fmt.Fprintln(r.stdout, strings.Repeat("─", 78))

	// Restore cursor after the frame is fully drawn.
	fmt.Fprint(r.stdout, "\033[?25h")
	return nil
}

// dashboardChannelTail returns the last n non-empty, non-heading lines of
// the team channel file, stripped of Markdown heading markers.
func dashboardChannelTail(repoPath string, n int) []string {
	content, err := readChannelFile(repoPath)
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		lines = append(lines, trimmed)
	}
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
