package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfigFilesRendersTemplates(t *testing.T) {
	root := t.TempDir()
	aomPath := filepath.Join(root, ".aom")
	if err := os.MkdirAll(aomPath, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	err := writeConfigFiles(aomPath, "my-app", root, "main", "my-app", "", nil)
	if err != nil {
		t.Fatalf("writeConfigFiles failed: %v", err)
	}

	projectData, err := os.ReadFile(filepath.Join(aomPath, "project.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(project.yaml) failed: %v", err)
	}
	if !strings.Contains(string(projectData), "name: my-app") {
		t.Fatalf("project.yaml = %q, want rendered project name", string(projectData))
	}

	agentsData, err := os.ReadFile(filepath.Join(aomPath, "agents.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(agents.yaml) failed: %v", err)
	}
	if !strings.Contains(string(agentsData), "backend-main:") {
		t.Fatalf("agents.yaml = %q, want baseline agent template", string(agentsData))
	}

	gitignoreData, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) failed: %v", err)
	}
	if !strings.Contains(string(gitignoreData), ".agent/") {
		t.Fatalf(".gitignore = %q, want .agent entry", string(gitignoreData))
	}
}

func TestWriteConfigFilesUsesCustomTemplateDir(t *testing.T) {
	root := t.TempDir()
	aomPath := filepath.Join(root, ".aom")
	templateDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(aomPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(.aom) failed: %v", err)
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(templates) failed: %v", err)
	}

	files := map[string]string{
		"project.yaml.tmpl":   "name: {{ .Name }}\nrepo: {{ .RepoPath }}\ndefault_branch: {{ .DefaultBranch }}\n\nruntime:\n  terminal: tmux\n  session_prefix: custom\n\ncontext:\n  state_dir: tasks\n  checkpoint_required: true\n",
		"agents.yaml.tmpl":    "roles: {}\nagents:\n  custom-main:\n    runtime: codex\n    role: custom\n    enabled: true\n",
		"resources.yaml.tmpl": "skills: {}\nmcp_servers: {}\nrole_bindings: {}\n",
		"policy.yaml.tmpl":    "policy:\n  deny_commands: []\n  require_approval: []\n  session_defaults:\n    approval_scope: per-session\n    yolo_mode: disabled\n  owner_exceptions:\n    enabled: true\n    log_required: true\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(templateDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", name, err)
		}
	}

	err := writeConfigFiles(aomPath, "my-app", root, "main", "my-app", templateDir, nil)
	if err != nil {
		t.Fatalf("writeConfigFiles failed: %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(aomPath, "agents.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(agents.yaml) failed: %v", err)
	}
	if !strings.Contains(string(agentsData), "custom-main:") {
		t.Fatalf("agents.yaml = %q, want custom template content", string(agentsData))
	}
}

func TestWriteConfigFilesAppendsAgentIgnoreWithoutOverwritingExistingGitignore(t *testing.T) {
	root := t.TempDir()
	aomPath := filepath.Join(root, ".aom")
	if err := os.MkdirAll(aomPath, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.gitignore) failed: %v", err)
	}

	if err := writeConfigFiles(aomPath, "my-app", root, "main", "my-app", "", nil); err != nil {
		t.Fatalf("writeConfigFiles failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "node_modules/") {
		t.Fatalf(".gitignore = %q, want existing content preserved", content)
	}
	if !strings.Contains(content, ".agent/") {
		t.Fatalf(".gitignore = %q, want .agent entry", content)
	}
	if strings.Count(content, ".agent/") != 1 {
		t.Fatalf(".gitignore = %q, want one .agent entry", content)
	}
}

func TestWriteConfigFilesFiltersSelectedAgents(t *testing.T) {
	root := t.TempDir()
	aomPath := filepath.Join(root, ".aom")
	if err := os.MkdirAll(aomPath, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := writeConfigFiles(aomPath, "my-app", root, "main", "my-app", "", []InitAgentSelection{
		{Name: "backend-main"},
		{Name: "reviewer-main"},
	}); err != nil {
		t.Fatalf("writeConfigFiles failed: %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(aomPath, "agents.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(agents.yaml) failed: %v", err)
	}
	content := string(agentsData)
	if !strings.Contains(content, "backend-main:") || !strings.Contains(content, "reviewer-main:") {
		t.Fatalf("agents.yaml = %q, want selected agents", content)
	}
	if strings.Contains(content, "orchestrator-main:") {
		t.Fatalf("agents.yaml = %q, do not want filtered-out agent", content)
	}
	if strings.Contains(content, "orchestrator:\n") {
		t.Fatalf("agents.yaml = %q, do not want unreferenced role", content)
	}
}

func TestWriteConfigFilesAddsInlineAgentUsingExistingRole(t *testing.T) {
	root := t.TempDir()
	aomPath := filepath.Join(root, ".aom")
	if err := os.MkdirAll(aomPath, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := writeConfigFiles(aomPath, "my-app", root, "main", "my-app", "", []InitAgentSelection{
		{Name: "backend-main"},
		{Name: "frontend-main", Role: "backend", Runtime: "claude", Inline: true},
	}); err != nil {
		t.Fatalf("writeConfigFiles failed: %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(aomPath, "agents.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(agents.yaml) failed: %v", err)
	}
	content := string(agentsData)
	if !strings.Contains(content, "frontend-main:") || !strings.Contains(content, "runtime: claude") || !strings.Contains(content, "role: backend") {
		t.Fatalf("agents.yaml = %q, want inline frontend agent", content)
	}
	if strings.Count(content, "backend:\n") != 1 {
		t.Fatalf("agents.yaml = %q, want reused backend role only once", content)
	}
}

func TestWriteConfigFilesAddsDefaultRoleForInlineAgent(t *testing.T) {
	root := t.TempDir()
	aomPath := filepath.Join(root, ".aom")
	if err := os.MkdirAll(aomPath, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := writeConfigFiles(aomPath, "my-app", root, "main", "my-app", "", []InitAgentSelection{
		{Name: "frontend-main", Role: "builder", Runtime: "CLAUDE", Inline: true},
	}); err != nil {
		t.Fatalf("writeConfigFiles failed: %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(aomPath, "agents.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(agents.yaml) failed: %v", err)
	}
	content := string(agentsData)
	if !strings.Contains(content, "frontend-main:") || !strings.Contains(content, "runtime: claude") || !strings.Contains(content, "role: builder") {
		t.Fatalf("agents.yaml = %q, want normalized inline frontend agent", content)
	}
	if !strings.Contains(content, "builder:") || !strings.Contains(content, "class: builder") || !strings.Contains(content, "worktree_mode: dedicated-writer") {
		t.Fatalf("agents.yaml = %q, want default inline role config", content)
	}
}

func TestParseInitAgentSelectionsRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{
			name:  "invalid bare name",
			input: []string{"frontend_main"},
			want:  `agent name "frontend_main" must be alphanumeric with hyphens only`,
		},
		{
			name:  "invalid runtime",
			input: []string{"frontend-main:builder:unknown"},
			want:  `agent "frontend-main" runtime "unknown" is not supported`,
		},
		{
			name:  "duplicate agent",
			input: []string{"frontend-main:builder:claude", "frontend-main:backend:codex"},
			want:  `agent "frontend-main" was selected more than once`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseInitAgentSelections(tt.input)
			if err == nil {
				t.Fatal("ParseInitAgentSelections returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestResolvePresetTemplateDirFindsTopLevelTemplate(t *testing.T) {
	path, err := resolvePresetTemplateDir("minimal")
	if err != nil {
		t.Fatalf("resolvePresetTemplateDir failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(path, "agents.yaml.tmpl")); err != nil {
		t.Fatalf("preset agents template missing: %v", err)
	}
}
