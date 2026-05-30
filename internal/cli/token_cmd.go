package cli

import "fmt"

// executeTokenUsage is a spike stub for F6 — Cost/Token Visibility.
// Neither the claude CLI nor the codex CLI currently exposes session-level token
// counts in a machine-readable format that AOM can parse at runtime. This command
// prints a summary of what each provider currently offers and how to access it.
func (r Runner) executeTokenUsage(_ []string) error {
	fmt.Fprintln(r.stdout, "Token Usage (spike — no automatic tracking yet)")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Provider    How to see token usage")
	fmt.Fprintln(r.stdout, "----------  ---------------------------------------------------------")
	fmt.Fprintln(r.stdout, "claude      claude.ai dashboard → Usage tab (web, per-account)")
	fmt.Fprintln(r.stdout, "            Claude CLI does not expose per-session token counts.")
	fmt.Fprintln(r.stdout, "codex       OpenAI platform.openai.com → Usage (web, per-account)")
	fmt.Fprintln(r.stdout, "            Codex CLI does not expose per-session token counts.")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Automatic tracking is not yet implemented. Contributions welcome:")
	fmt.Fprintln(r.stdout, "  If your provider exposes usage in CLI output or a log file,")
	fmt.Fprintln(r.stdout, "  open an issue with a sample of the output format.")
	return nil
}
