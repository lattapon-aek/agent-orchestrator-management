package provider

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type codexProvider struct{}

func (p *codexProvider) Name() string            { return "codex" }
func (p *codexProvider) IdentityFilename() string { return "AGENTS.md" }

func (p *codexProvider) LaunchShellSpec(spec LaunchSpec, lookPath func(string) (string, error)) (ShellSpec, error) {
	if _, err := lookPath("codex"); err != nil {
		return ShellSpec{}, fmt.Errorf("real launch for runtime %q requires the %q CLI in PATH", "codex", "codex")
	}
	preamble := []string{"export AOM_RUNTIME=codex"}
	if len(spec.DenyCommands) > 0 {
		preamble = append(preamble, buildCodexWrapperPreamble(spec.SessionID, spec.DenyCommands)...)
	}
	var execCmd string
	if spec.AgentSessionID != "" {
		execCmd = fmt.Sprintf("exec codex resume %s --sandbox workspace-write -a never", spec.AgentSessionID)
	} else {
		execCmd = "exec codex --sandbox workspace-write -a never"
	}
	if spec.Model != "" {
		execCmd += " -m " + spec.Model
	}
	return ShellSpec{
		Preamble: preamble,
		ExecCmd:  execCmd,
	}, nil
}

// buildCodexWrapperPreamble generates preamble statements that create lightweight
// shell wrapper scripts blocking each denied command. The wrapper bin dir is
// prepended to PATH before exec, so codex and its subprocesses intercept the
// blocked commands at the shell level.
//
// Each deny_command entry is split on the first space; only the base command
// (first word) gets a wrapper. Duplicate base commands are skipped. The wrapper
// always exits 1 with a policy message — no passthrough — because codex has no
// native partial-command-blocking flag.
//
// The bin dir is session-scoped under /tmp so the OS cleans it up on reboot.
func buildCodexWrapperPreamble(sessionID string, denyCommands []string) []string {
	binDir := fmt.Sprintf("/tmp/aom-policy-%s/bin", sessionID)
	stmts := []string{
		fmt.Sprintf(`mkdir -p "%s"`, binDir),
	}
	seen := make(map[string]bool)
	for _, rawCmd := range denyCommands {
		cmd := strings.TrimSpace(rawCmd)
		if cmd == "" {
			continue
		}
		baseCmd := strings.Fields(cmd)[0]
		if seen[baseCmd] {
			continue
		}
		seen[baseCmd] = true
		// printf format: double quotes wrap the format so \n and \" are shell-processed.
		// \n is kept as two chars by the shell and interpreted by printf as newline.
		// \" inside double quotes → literal " in the format string passed to printf.
		// No single quotes appear here, keeping compatibility with the outer sh -lc '...' wrapper.
		stmts = append(stmts,
			fmt.Sprintf(
				`printf "#!/bin/sh\necho \"AOM policy: %s blocked by project policy\" >&2\nexit 1\n" > "%s/%s" && chmod +x "%s/%s"`,
				baseCmd, binDir, baseCmd, binDir, baseCmd,
			),
		)
	}
	stmts = append(stmts, fmt.Sprintf(`export PATH="%s:$PATH"`, binDir))
	return stmts
}

func (p *codexProvider) ResumeInfo() ResumeInfo {
	return ResumeInfo{
		Supported:     true,
		FreshExample:  "codex --sandbox workspace-write",
		ResumeExample: "codex resume <session-id> --sandbox workspace-write",
	}
}

func (p *codexProvider) MCPConfigStyle() MCPStyle                  { return MCPStyleJSONFile }
func (p *codexProvider) PolicyEnforcementLevel() PolicyEnforcement { return PolicyEnforcementWrapperScript }
// StartupDialogResponse returns "1" to accept codex's directory trust dialog
// ("1. Yes, continue") shown on fresh starts in new or untrusted directories.
func (p *codexProvider) StartupDialogResponse() string { return "1" }

func (p *codexProvider) ModelHint() string {
	return "Known slugs: gpt-5.5, gpt-5.4, gpt-5.4-mini, gpt-5.3-codex, gpt-5.2. " +
		"Full list cached at ~/.codex/models_cache.json (auto-refreshed by codex on startup)."
}

func (p *codexProvider) KnownModels() []string {
	return []string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex", "gpt-5.2"}
}

func (p *codexProvider) NativeSessionDetection() *NativeSessionStrategy {
	return &NativeSessionStrategy{DetectFn: codexSessionAfterSpawn}
}

// codexSessionAfterSpawn polls ~/.codex/logs_2.sqlite for the first thread_id
// that appears at or after spawnedAt. Returns the session UUID on success, or
// an empty string if none is found within timeout.
func codexSessionAfterSpawn(_ string, spawnedAt time.Time, timeout time.Duration) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	dbPath := filepath.Join(home, ".codex", "logs_2.sqlite")

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if id := queryNewestCodexSession(dbPath, spawnedAt); id != "" {
			return id, nil
		}
		time.Sleep(time.Second)
	}
	return "", nil
}

// queryNewestCodexSession opens codex's logs_2.sqlite read-only and returns the
// first thread_id logged at or after spawnedAt, or "" if none found yet.
func queryNewestCodexSession(dbPath string, spawnedAt time.Time) string {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_busy_timeout=1000")
	if err != nil {
		return ""
	}
	defer db.Close()

	var id string
	if err := db.QueryRow(
		`SELECT DISTINCT thread_id FROM logs
		 WHERE thread_id IS NOT NULL AND ts >= ?
		 ORDER BY ts ASC, ts_nanos ASC LIMIT 1`,
		spawnedAt.Unix(),
	).Scan(&id); err != nil {
		return ""
	}
	return id
}
