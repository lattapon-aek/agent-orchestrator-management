package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/config"
)

func seedAgentProfiles(aomPath string, cfg *config.ProjectConfig) error {
	if cfg == nil {
		return fmt.Errorf("project config is required")
	}

	for agentName, agentCfg := range cfg.Agents.Agents {
		roleCfg, ok := cfg.Agents.Roles[agentCfg.Role]
		if !ok {
			return fmt.Errorf("agent %q references unknown role %q", agentName, agentCfg.Role)
		}

		profilePath := filepath.Join(aomPath, "agents", agentName, "profile.md")
		if _, err := os.Stat(profilePath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat agent profile %q: %w", profilePath, err)
		}

		if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
			return fmt.Errorf("create agent profile dir %q: %w", filepath.Dir(profilePath), err)
		}

		content := renderAgentProfileMarkdown(agentName, agentCfg.Role, agentCfg.Runtime, roleCfg.Class)
		if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write agent profile %q: %w", profilePath, err)
		}
	}

	return nil
}

func renderAgentProfileMarkdown(agentName, roleName, runtimeName, roleClass string) string {
	return fmt.Sprintf(`# Agent Identity

## Identity
- Agent: %s
- Role: %s
- Runtime: %s

## Responsibilities
- %s

## Working Protocol
- Always begin by reading .agent/task.md and .agent/state.md
- Update .agent/state.md as work progresses
- On completion: write .agent/handoff.md and append handoff.prepared or task.completed to .agent/log.md

## Constraints
- Stay within the current task scope
- Do not modify .agent/index.md or .agent/log.md because those artifacts are AOM-owned
`,
		agentName,
		roleName,
		runtimeName,
		defaultResponsibility(roleClass),
	)
}

func defaultResponsibility(roleClass string) string {
	switch strings.TrimSpace(roleClass) {
	case "reviewer":
		return "Review implementation work against the task artifacts and record actionable findings"
	case "orchestrator":
		return "Coordinate task flow, handoffs, and next actions according to the project artifacts"
	default:
		return "Implement the assigned task work according to the task artifacts and current session state"
	}
}
