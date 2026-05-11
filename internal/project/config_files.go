package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/config"
	"gopkg.in/yaml.v3"
)

type baselineProjectConfig struct {
	Project   config.ProjectFile
	Agents    config.AgentsFile
	Resources config.ResourcesFile
	Policy    config.PolicyFile
}

func baselineConfig(name, repoPath, defaultBranch, sessionPrefix string) baselineProjectConfig {
	return baselineProjectConfig{
		Project: config.ProjectFile{
			Name:          name,
			Repo:          repoPath,
			DefaultBranch: defaultBranch,
			Runtime: config.RuntimeConfig{
				Terminal:      "tmux",
				SessionPrefix: sessionPrefix,
			},
			Context: config.ContextConfig{
				StateDir:           ".agent",
				CheckpointRequired: true,
			},
		},
		Agents: config.AgentsFile{
			Roles: map[string]config.RoleConfig{
				"orchestrator": {
					Class:                 "orchestrator",
					WorktreeMode:          "read-only",
					CheckpointExpectation: "required",
					DefaultSessionMode:    "interactive",
				},
				"backend": {
					Class:                 "builder",
					WorktreeMode:          "dedicated-writer",
					CheckpointExpectation: "required",
					DefaultSessionMode:    "interactive",
				},
				"reviewer": {
					Class:                 "reviewer",
					WorktreeMode:          "read-only",
					CheckpointExpectation: "required",
					DefaultSessionMode:    "interactive",
				},
			},
			Agents: map[string]config.AgentConfig{
				"orchestrator-main": {
					Runtime: "claude",
					Role:    "orchestrator",
					Enabled: true,
				},
				"backend-main": {
					Runtime: "codex",
					Role:    "backend",
					Enabled: true,
				},
				"reviewer-main": {
					Runtime: "claude",
					Role:    "reviewer",
					Enabled: true,
				},
			},
		},
		Resources: config.ResourcesFile{
			Skills:       map[string]config.SkillConfig{},
			MCPServers:   map[string]config.MCPServerConfig{},
			RoleBindings: map[string]config.RoleBindingConfig{},
		},
		Policy: config.PolicyFile{
			Policy: config.PolicyConfig{
				DenyCommands: []string{
					"rm -rf",
					"git push --force",
					"curl * | sh",
					"npm publish",
					"terraform apply",
				},
				RequireApproval: []string{
					"delete file",
					"database migration",
					"deploy",
					"read secrets",
					"network access",
				},
				SessionDefaults: config.SessionDefaultsConfig{
					ApprovalScope: "per-session",
					YoloMode:      "disabled",
				},
				OwnerExceptions: config.OwnerExceptionsConfig{
					Enabled:     true,
					LogRequired: true,
				},
			},
		},
	}
}

func writeConfigFiles(aomPath string, cfg baselineProjectConfig) error {
	files := map[string]any{
		"project.yaml":   cfg.Project,
		"agents.yaml":    cfg.Agents,
		"resources.yaml": cfg.Resources,
		"policy.yaml":    cfg.Policy,
	}

	for name, value := range files {
		data, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", name, err)
		}

		path := filepath.Join(aomPath, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	return nil
}
