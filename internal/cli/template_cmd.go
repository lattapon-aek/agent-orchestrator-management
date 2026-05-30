package cli

import (
	"fmt"
	"strings"
)

// taskTemplate defines a reusable task configuration preset.
type taskTemplate struct {
	Name        string
	Description string
	Mode        string
	Steps       []taskTemplateStep
}

// taskTemplateStep is one step entry within a task template.
type taskTemplateStep struct {
	StepType string
	Title    string
}

// builtinTaskTemplates is the set of shipped templates accessible via --template.
var builtinTaskTemplates = map[string]taskTemplate{
	"small-fix": {
		Name:        "small-fix",
		Description: "Targeted bug fix with narrow scope",
		Mode:        "Bugfix",
		Steps: []taskTemplateStep{
			{StepType: "investigate", Title: "Investigate root cause"},
			{StepType: "implement", Title: "Apply fix"},
		},
	},
	"feature-standard": {
		Name:        "feature-standard",
		Description: "Standard feature with design, implementation, and tests",
		Mode:        "Direct",
		Steps: []taskTemplateStep{
			{StepType: "design", Title: "Design approach"},
			{StepType: "implement", Title: "Implement feature"},
			{StepType: "test", Title: "Write and run tests"},
		},
	},
	"risky-change": {
		Name:        "risky-change",
		Description: "High-risk change with requirements, design, implementation, and explicit review",
		Mode:        "Requirements-first",
		Steps: []taskTemplateStep{
			{StepType: "requirements", Title: "Document requirements"},
			{StepType: "design", Title: "Design solution"},
			{StepType: "implement", Title: "Implement"},
			{StepType: "test", Title: "Test and verify"},
			{StepType: "review", Title: "Peer review"},
		},
	},
	"qa-pass": {
		Name:        "qa-pass",
		Description: "Quality assurance check and sign-off pass",
		Mode:        "Direct",
		Steps: []taskTemplateStep{
			{StepType: "qa-check", Title: "Run QA checks"},
			{StepType: "review", Title: "Sign off"},
		},
	},
	"research-spike": {
		Name:        "research-spike",
		Description: "Time-boxed research and exploration",
		Mode:        "Direct",
		Steps: []taskTemplateStep{
			{StepType: "research", Title: "Research and document findings"},
		},
	},
}

// builtinTemplateOrder preserves display ordering for aom task templates.
var builtinTemplateOrder = []string{
	"small-fix",
	"feature-standard",
	"risky-change",
	"qa-pass",
	"research-spike",
}

// executeTaskTemplates lists available built-in task templates.
func (r Runner) executeTaskTemplates(_ []string) error {
	fmt.Fprintln(r.stdout, "Available task templates:")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "  %-20s  %-12s  %s\n", "NAME", "MODE", "DESCRIPTION")
	fmt.Fprintf(r.stdout, "  %-20s  %-12s  %s\n",
		strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 40))
	for _, name := range builtinTemplateOrder {
		tpl := builtinTaskTemplates[name]
		fmt.Fprintf(r.stdout, "  %-20s  %-12s  %s\n", tpl.Name, tpl.Mode, tpl.Description)
	}
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Usage: aom task create \"<title>\" --template <name>")
	return nil
}

// resolveTaskTemplate looks up a built-in template by name.
func resolveTaskTemplate(name string) (*taskTemplate, error) {
	t, ok := builtinTaskTemplates[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("unknown template %q — run 'aom task templates' to list available templates", name)
	}
	return &t, nil
}
