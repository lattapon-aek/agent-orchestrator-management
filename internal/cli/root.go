package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/agent"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/app"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/artifact"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/plan"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/project"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/session"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/step"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/task"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/tmux"
)

var newApp = app.New

// Runner executes top-level CLI behavior.
type Runner struct {
	app    *app.App
	stdout io.Writer
	stderr io.Writer
}

// Execute runs the AOM CLI using the provided arguments and streams.
func Execute(args []string, stdout, stderr io.Writer) error {
	r := Runner{
		app:    newApp(),
		stdout: stdout,
		stderr: stderr,
	}

	return r.Execute(args)
}

// Execute dispatches a command line invocation.
func (r Runner) Execute(args []string) error {
	_ = r.app

	if len(args) == 0 {
		r.printHelp()
		return nil
	}

	switch args[0] {
	case "help", "--help", "-h":
		r.printHelp()
		return nil
	case "attach":
		return r.executeAttach(args[1:])
	case "capture":
		return r.executeCapture(args[1:])
	case "open":
		return r.executeOpen(args[1:])
	case "plan":
		return r.executePlan(args[1:])
	case "step":
		return r.executeStep(args[1:])
	case "session":
		return r.executeSession(args[1:])
	case "status":
		return r.executeStatus(args[1:])
	case "task":
		return r.executeTask(args[1:])
	case "project":
		return r.executeProject(args[1:])
	default:
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}
}

func (r Runner) executeTask(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task subcommand is required")
	}

	switch args[0] {
	case "create":
		return r.executeTaskCreate(args[1:])
	case "update":
		return r.executeTaskUpdate(args[1:])
	case "close":
		return r.executeTaskClose(args[1:])
	case "show":
		return r.executeTaskShow(args[1:])
	default:
		return fmt.Errorf("unknown task command %q", strings.Join(args, " "))
	}
}

func (r Runner) executePlan(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("work description is required")
	}

	params := planParams{workDescription: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--create":
			params.createTask = true
		case "--mode":
			i++
			if i >= len(args) {
				return fmt.Errorf("--mode requires a value")
			}
			params.mode = args[i]
		case "--role":
			i++
			if i >= len(args) {
				return fmt.Errorf("--role requires a value")
			}
			params.role = args[i]
		case "--agent":
			i++
			if i >= len(args) {
				return fmt.Errorf("--agent requires a value")
			}
			params.agent = args[i]
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	planResult, err := r.app.Planner.Build(plan.Params{
		WorkDescription: params.workDescription,
		Mode:            params.mode,
		PreferredRole:   params.role,
		PreferredAgent:  params.agent,
		Agents:          result.Agents,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Plan")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Work: %s\n", params.workDescription)
	fmt.Fprintf(r.stdout, "Mode: %s\n", planResult.Mode)
	fmt.Fprintf(r.stdout, "Recommended role: %s\n", emptyFallback(planResult.RecommendedRole))
	fmt.Fprintf(r.stdout, "Recommended agent: %s\n", emptyFallback(planResult.RecommendedAgent))
	fmt.Fprintln(r.stdout, "Proposed steps:")
	for i, item := range planResult.Steps {
		fmt.Fprintf(
			r.stdout,
			"  %d. type=%s | title=%s | role=%s | agent=%s\n",
			i+1,
			item.Type,
			item.Title,
			emptyFallback(item.RoleName),
			emptyFallback(item.AgentName),
		)
	}
	fmt.Fprintf(r.stdout, "Recommended next action: %s\n", planResult.RecommendedNextAction)

	if !params.createTask {
		return nil
	}

	taskService, sqlDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	createResult, err := taskService.CreateFromPlan(task.CreateParams{
		ProjectID:      result.Project.ID,
		Title:          params.workDescription,
		Mode:           planResult.Mode,
		PreferredRole:  planResult.RecommendedRole,
		PreferredAgent: planResult.RecommendedAgent,
	}, buildPlanStepSeeds(planResult.Steps))
	if err != nil {
		return err
	}

	if err := r.syncTaskArtifacts(result, createResult.Task.ID, artifact.Event{
		Type:        "task.created",
		Actor:       "operator",
		Summary:     fmt.Sprintf("Task created from plan in %s mode", createResult.Task.Mode),
		StateEffect: fmt.Sprintf("Task %s", createResult.Task.Status),
	}, true); err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Task created from plan")
	fmt.Fprintf(r.stdout, "Task: %s\n", createResult.Task.ID)
	fmt.Fprintf(r.stdout, "Status: %s\n", createResult.Task.Status)
	fmt.Fprintf(r.stdout, "Steps created: %d\n", len(createResult.Steps))

	return nil
}

type planParams struct {
	workDescription string
	mode            string
	role            string
	agent           string
	createTask      bool
}

func (r Runner) executeStep(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("step subcommand is required")
	}

	switch args[0] {
	case "list":
		return r.executeStepList(args[1:])
	case "update":
		return r.executeStepUpdate(args[1:])
	default:
		return fmt.Errorf("unknown step command %q", strings.Join(args, " "))
	}
}

func (r Runner) executeSession(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session subcommand is required")
	}

	switch args[0] {
	case "spawn":
		return r.executeSessionSpawn(args[1:])
	case "list":
		return r.executeSessionList(args[1:])
	case "show":
		return r.executeSessionShow(args[1:])
	default:
		return fmt.Errorf("unknown session command %q", strings.Join(args, " "))
	}
}

func (r Runner) executeProject(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project subcommand is required")
	}

	switch args[0] {
	case "init":
		return r.executeProjectInit(args[1:])
	default:
		return fmt.Errorf("unknown project command %q", strings.Join(args, " "))
	}
}

func (r Runner) executeProjectInit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project name is required")
	}

	params := projectInitParams{
		name: args[0],
	}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--repo":
			i++
			if i >= len(args) {
				return fmt.Errorf("--repo requires a value")
			}
			params.repo = args[i]
		case "--default-branch":
			i++
			if i >= len(args) {
				return fmt.Errorf("--default-branch requires a value")
			}
			params.defaultBranch = args[i]
		case "--session-prefix":
			i++
			if i >= len(args) {
				return fmt.Errorf("--session-prefix requires a value")
			}
			params.sessionPrefix = args[i]
		case "--template":
			i++
			if i >= len(args) {
				return fmt.Errorf("--template requires a value")
			}
			params.templateName = args[i]
		case "--template-dir":
			i++
			if i >= len(args) {
				return fmt.Errorf("--template-dir requires a value")
			}
			params.templateDir = args[i]
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	if strings.TrimSpace(params.repo) == "" {
		return fmt.Errorf("--repo is required")
	}

	result, err := r.app.Projects.Init(params.toInitParams())
	if err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Project initialized")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Name: %s\n", result.ProjectName)
	fmt.Fprintf(r.stdout, "Repo: %s\n", result.RepoPath)
	fmt.Fprintf(r.stdout, "AOM: %s\n", result.AOMPath)
	fmt.Fprintf(r.stdout, "DB: %s\n", result.DBPath)
	fmt.Fprintf(r.stdout, "Config: %s\n", filepath.Join(result.AOMPath, "project.yaml"))

	return nil
}

type projectInitParams struct {
	name          string
	repo          string
	defaultBranch string
	sessionPrefix string
	templateName  string
	templateDir   string
}

func (p projectInitParams) toInitParams() project.InitParams {
	return project.InitParams{
		Name:          p.name,
		RepoPath:      p.repo,
		DefaultBranch: p.defaultBranch,
		SessionPrefix: p.sessionPrefix,
		TemplateName:  p.templateName,
		TemplateDir:   p.templateDir,
	}
}

func (r Runner) executeOpen(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("open does not accept positional arguments in the current milestone")
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	workspace, err := r.app.Tmux.EnsureWorkspace(result.SessionPrefix, result.Project.RepoPath)
	if err != nil {
		return fmt.Errorf("ensure tmux workspace: %w", err)
	}

	sessions, err := r.loadProjectSessions(result)
	if err != nil {
		return err
	}

	taskCount, err := r.loadTaskCount(result)
	if err != nil {
		return err
	}

	taskViews, err := r.loadTaskViews(result)
	if err != nil {
		return err
	}

	r.printProjectSummary("Project opened", result, workspace, sessions, taskCount, taskViews)
	return nil
}

func (r Runner) executeStatus(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("status does not accept positional arguments in the current milestone")
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	sessions, err := r.loadProjectSessions(result)
	if err != nil {
		return err
	}

	taskCount, err := r.loadTaskCount(result)
	if err != nil {
		return err
	}

	taskViews, err := r.loadTaskViews(result)
	if err != nil {
		return err
	}

	r.printProjectSummary("Project status", result, nil, sessions, taskCount, taskViews)
	return nil
}

func (r Runner) executeTaskCreate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task title is required")
	}

	params := taskCreateParams{title: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			i++
			if i >= len(args) {
				return fmt.Errorf("--mode requires a value")
			}
			params.mode = args[i]
		case "--role":
			i++
			if i >= len(args) {
				return fmt.Errorf("--role requires a value")
			}
			params.role = args[i]
		case "--agent":
			i++
			if i >= len(args) {
				return fmt.Errorf("--agent requires a value")
			}
			params.agent = args[i]
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	taskService, sqlDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	createResult, err := taskService.Create(task.CreateParams{
		ProjectID:      result.Project.ID,
		Title:          params.title,
		Mode:           params.mode,
		PreferredRole:  params.role,
		PreferredAgent: params.agent,
	})
	if err != nil {
		return err
	}

	if err := r.syncTaskArtifacts(result, createResult.Task.ID, artifact.Event{
		Type:        "task.created",
		Actor:       "operator",
		Summary:     fmt.Sprintf("Task created in %s mode", createResult.Task.Mode),
		StateEffect: fmt.Sprintf("Task %s", createResult.Task.Status),
	}, true); err != nil {
		return err
	}

	recommendedNext := "confirm the initial step and move the task to Ready"
	if createResult.Task.PreferredRole != "" || createResult.Task.PreferredAgent != "" {
		recommendedNext = "confirm the initial step owner and move the task to Ready"
	}

	fmt.Fprintln(r.stdout, "Task created")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Task: %s\n", createResult.Task.ID)
	fmt.Fprintf(r.stdout, "Title: %s\n", createResult.Task.Title)
	fmt.Fprintf(r.stdout, "Mode: %s\n", createResult.Task.Mode)
	fmt.Fprintf(r.stdout, "Status: %s\n", createResult.Task.Status)
	fmt.Fprintf(r.stdout, "Initial steps: %d\n", len(createResult.Steps))
	fmt.Fprintf(r.stdout, "Recommended next step: %s\n", recommendedNext)

	return nil
}

type taskCreateParams struct {
	title string
	mode  string
	role  string
	agent string
}

func (r Runner) executeTaskShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task identifier is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("task show does not accept extra positional arguments in the current milestone")
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	taskService, taskDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return err
	}
	defer taskDB.Close()

	taskRecord, err := taskService.Get(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}
	if taskRecord == nil {
		return fmt.Errorf("task %q not found", strings.TrimSpace(args[0]))
	}

	stepService, stepDB, err := r.app.OpenStepService(result.DBPath)
	if err != nil {
		return err
	}
	defer stepDB.Close()

	steps, err := stepService.ListByTask(taskRecord.ID)
	if err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Task")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "ID: %s\n", taskRecord.ID)
	fmt.Fprintf(r.stdout, "Title: %s\n", taskRecord.Title)
	fmt.Fprintf(r.stdout, "Mode: %s\n", taskRecord.Mode)
	fmt.Fprintf(r.stdout, "Status: %s\n", taskRecord.Status)
	fmt.Fprintf(r.stdout, "Preferred role: %s\n", emptyFallback(taskRecord.PreferredRole))
	fmt.Fprintf(r.stdout, "Preferred agent: %s\n", emptyFallback(taskRecord.PreferredAgent))
	fmt.Fprintf(r.stdout, "Steps: %d\n", len(steps))
	fmt.Fprintf(r.stdout, "Recommended next action: %s\n", recommendTaskAction(taskRecord.Status, steps))

	return nil
}

func (r Runner) executeTaskUpdate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task identifier is required")
	}

	params := taskUpdateParams{id: strings.TrimSpace(args[0])}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			i++
			if i >= len(args) {
				return fmt.Errorf("--mode requires a value")
			}
			params.mode = args[i]
		case "--status":
			i++
			if i >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			params.status = args[i]
		case "--role":
			i++
			if i >= len(args) {
				return fmt.Errorf("--role requires a value")
			}
			params.role = args[i]
		case "--agent":
			i++
			if i >= len(args) {
				return fmt.Errorf("--agent requires a value")
			}
			params.agent = args[i]
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	taskService, sqlDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	record, err := taskService.Update(params.id, task.UpdateParams{
		Mode:           params.mode,
		Status:         params.status,
		PreferredRole:  params.role,
		PreferredAgent: params.agent,
	})
	if err != nil {
		return err
	}

	if err := r.syncTaskArtifacts(result, record.ID, artifact.Event{
		Type:        mapTaskEventType(params.status, params.mode),
		Actor:       "operator",
		Summary:     fmt.Sprintf("Task updated to mode=%s status=%s", record.Mode, record.Status),
		StateEffect: fmt.Sprintf("Task %s", record.Status),
	}, false); err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Task updated")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Task: %s\n", record.ID)
	fmt.Fprintf(r.stdout, "Mode: %s\n", record.Mode)
	fmt.Fprintf(r.stdout, "Status: %s\n", record.Status)
	fmt.Fprintf(r.stdout, "Preferred role: %s\n", emptyFallback(record.PreferredRole))
	fmt.Fprintf(r.stdout, "Preferred agent: %s\n", emptyFallback(record.PreferredAgent))

	return nil
}

type taskUpdateParams struct {
	id     string
	mode   string
	status string
	role   string
	agent  string
}

func (r Runner) executeTaskClose(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task identifier is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("task close does not accept extra positional arguments in the current milestone")
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	taskService, sqlDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	record, err := taskService.Close(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}

	if err := r.syncTaskArtifacts(result, record.ID, artifact.Event{
		Type:        "task.closed",
		Actor:       "operator",
		Summary:     "Task closed explicitly by operator",
		StateEffect: fmt.Sprintf("Task %s", record.Status),
	}, false); err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Task closed")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Task: %s\n", record.ID)
	fmt.Fprintf(r.stdout, "Status: %s\n", record.Status)

	return nil
}

func (r Runner) executeStepList(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task identifier is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("step list does not accept extra positional arguments in the current milestone")
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	taskService, taskDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return err
	}
	defer taskDB.Close()

	taskRecord, err := taskService.Get(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}
	if taskRecord == nil {
		return fmt.Errorf("task %q not found", strings.TrimSpace(args[0]))
	}

	stepService, stepDB, err := r.app.OpenStepService(result.DBPath)
	if err != nil {
		return err
	}
	defer stepDB.Close()

	steps, err := stepService.ListByTask(taskRecord.ID)
	if err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Steps")
	fmt.Fprintln(r.stdout, "")
	if len(steps) == 0 {
		fmt.Fprintf(r.stdout, "No steps for %s\n", taskRecord.ID)
		return nil
	}

	for _, item := range steps {
		fmt.Fprintf(
			r.stdout,
			"  - %s | type=%s | title=%s | role=%s | agent=%s | status=%s | dependencies=%s\n",
			item.ID,
			item.StepType,
			item.Title,
			emptyFallback(item.RoleName),
			emptyFallback(item.AgentName),
			item.Status,
			formatDependencies(item.Dependencies),
		)
	}

	return nil
}

func (r Runner) executeStepUpdate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("step identifier is required")
	}

	params := stepUpdateParams{id: strings.TrimSpace(args[0])}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--status":
			i++
			if i >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			params.status = args[i]
		case "--role":
			i++
			if i >= len(args) {
				return fmt.Errorf("--role requires a value")
			}
			params.role = args[i]
		case "--agent":
			i++
			if i >= len(args) {
				return fmt.Errorf("--agent requires a value")
			}
			params.agent = args[i]
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	stepService, sqlDB, err := r.app.OpenStepService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	record, err := stepService.Update(params.id, step.UpdateParams{
		Status:    params.status,
		RoleName:  params.role,
		AgentName: params.agent,
	})
	if err != nil {
		return err
	}

	if err := r.syncTaskArtifacts(result, record.TaskID, artifact.Event{
		Type:        mapStepEventType(record.Status),
		Actor:       "operator",
		StepID:      record.ID,
		Summary:     fmt.Sprintf("Step updated to %s", record.Status),
		StateEffect: fmt.Sprintf("Step %s", record.Status),
	}, false); err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Step updated")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Step: %s\n", record.ID)
	fmt.Fprintf(r.stdout, "Status: %s\n", record.Status)
	fmt.Fprintf(r.stdout, "Role: %s\n", emptyFallback(record.RoleName))
	fmt.Fprintf(r.stdout, "Agent: %s\n", emptyFallback(record.AgentName))

	return nil
}

type stepUpdateParams struct {
	id     string
	status string
	role   string
	agent  string
}

func (r Runner) executeSessionSpawn(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("agent name is required")
	}

	params := sessionSpawnParams{
		agentName: strings.TrimSpace(args[0]),
	}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--task":
			i++
			if i >= len(args) {
				return fmt.Errorf("--task requires a value")
			}
			params.taskID = strings.TrimSpace(args[i])
		case "--step":
			i++
			if i >= len(args) {
				return fmt.Errorf("--step requires a value")
			}
			params.stepID = strings.TrimSpace(args[i])
		case "--mock":
			params.mockRuntime = true
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	agentRecord, err := findAgent(result.Agents, params.agentName)
	if err != nil {
		return err
	}

	var taskRecord *task.Record
	if params.taskID != "" {
		taskRecord, err = r.loadTaskByID(result, params.taskID)
		if err != nil {
			return err
		}
		if taskRecord == nil {
			return fmt.Errorf("task %q not found", params.taskID)
		}
	}

	var stepRecord *step.Record
	if params.stepID != "" {
		if taskRecord == nil {
			return fmt.Errorf("--step requires --task in the current milestone")
		}
		stepRecord, err = r.loadStepByID(result, params.stepID)
		if err != nil {
			return err
		}
		if stepRecord == nil {
			return fmt.Errorf("step %q not found", params.stepID)
		}
		if stepRecord.TaskID != taskRecord.ID {
			return fmt.Errorf("step %q does not belong to task %q", stepRecord.ID, taskRecord.ID)
		}
	}

	workspace, err := r.app.Tmux.EnsureWorkspace(result.SessionPrefix, result.Project.RepoPath)
	if err != nil {
		return fmt.Errorf("ensure tmux workspace: %w", err)
	}

	sessionService, sqlDB, err := r.app.OpenSessionService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	record, err := sessionService.Create(session.CreateParams{
		ProjectID:       result.Project.ID,
		AgentID:         agentRecord.ID,
		AgentName:       agentRecord.Name,
		RoleName:        agentRecord.Role,
		TaskID:          params.taskID,
		Runtime:         agentRecord.Runtime,
		Status:          "Booting",
		RepoPath:        result.Project.RepoPath,
		WorktreePath:    result.Project.RepoPath,
		TmuxSessionName: workspace.Name,
	})
	if err != nil {
		return err
	}

	paneBinding, err := r.app.Tmux.CreatePane(workspace.Target, result.Project.RepoPath, sessionLaunchCommand(*record, params.mockRuntime))
	if err != nil {
		return err
	}

	record.Status = "Idle"
	record.TmuxWindow = paneBinding.WindowID
	record.TmuxPane = paneBinding.PaneID
	record.TmuxSessionName = workspace.Name

	record, err = sessionService.Save(*record)
	if err != nil {
		return err
	}

	if err := r.app.Tmux.AnnotatePane(record.TmuxPane, map[string]string{
		"@aom_session_id": record.ID,
		"@aom_agent":      record.AgentName,
		"@aom_role":       record.RoleName,
	}); err != nil {
		return err
	}

	if taskRecord != nil {
		if err := r.syncTaskArtifactsWithSession(result, taskRecord.ID, artifact.Event{
			Type:        "session.created",
			Actor:       "aom",
			StepID:      params.stepID,
			SessionID:   record.ID,
			Summary:     fmt.Sprintf("Session spawned for %s using %s launch mode", agentRecord.Name, launchModeLabel(params.mockRuntime)),
			StateEffect: fmt.Sprintf("Session %s", record.Status),
		}, false, record); err != nil {
			return err
		}
	}

	fmt.Fprintln(r.stdout, "Session spawned")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Session: %s\n", record.ID)
	fmt.Fprintf(r.stdout, "Agent: %s\n", record.AgentName)
	fmt.Fprintf(r.stdout, "Role: %s\n", record.RoleName)
	if taskRecord != nil {
		fmt.Fprintf(r.stdout, "Task: %s\n", taskRecord.ID)
	}
	if stepRecord != nil {
		fmt.Fprintf(r.stdout, "Step: %s\n", stepRecord.ID)
	}
	fmt.Fprintf(r.stdout, "Runtime: %s\n", record.Runtime)
	fmt.Fprintf(r.stdout, "Launch mode: %s\n", launchModeLabel(params.mockRuntime))
	fmt.Fprintf(r.stdout, "Workspace: %s\n", workspace.Target)
	fmt.Fprintf(r.stdout, "Window: %s\n", record.TmuxWindow)
	fmt.Fprintf(r.stdout, "Pane: %s\n", record.TmuxPane)

	return nil
}

type sessionSpawnParams struct {
	agentName   string
	taskID      string
	stepID      string
	mockRuntime bool
}

func (r Runner) executeSessionList(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("session list does not accept positional arguments in the current milestone")
	}

	result, err := r.app.Projects.Open(".")
	if err != nil {
		return err
	}

	sessionService, sqlDB, err := r.app.OpenSessionService(result.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	sessions, err := sessionService.ListByProject(result.Project.ID)
	if err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Sessions")
	fmt.Fprintln(r.stdout, "")
	if len(sessions) == 0 {
		fmt.Fprintln(r.stdout, "No sessions")
		return nil
	}

	for _, item := range sessions {
		fmt.Fprintf(
			r.stdout,
			"  - %s | agent=%s | role=%s | task=%s | runtime=%s | status=%s | tmux=%s %s %s\n",
			item.ID,
			item.AgentName,
			item.RoleName,
			emptyFallback(item.TaskID),
			item.Runtime,
			item.Status,
			item.TmuxSessionName,
			item.TmuxWindow,
			item.TmuxPane,
		)
	}

	return nil
}

func (r Runner) executeSessionShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session identifier is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("session show does not accept extra positional arguments in the current milestone")
	}

	sessionRecord, err := r.loadSessionByIdentifier(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "Session")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "ID: %s\n", sessionRecord.ID)
	fmt.Fprintf(r.stdout, "Agent: %s\n", sessionRecord.AgentName)
	fmt.Fprintf(r.stdout, "Role: %s\n", sessionRecord.RoleName)
	fmt.Fprintf(r.stdout, "Task: %s\n", emptyFallback(sessionRecord.TaskID))
	fmt.Fprintf(r.stdout, "Runtime: %s\n", sessionRecord.Runtime)
	fmt.Fprintf(r.stdout, "Status: %s\n", sessionRecord.Status)
	fmt.Fprintf(r.stdout, "Repo: %s\n", sessionRecord.RepoPath)
	fmt.Fprintf(r.stdout, "Worktree: %s\n", sessionRecord.WorktreePath)
	fmt.Fprintf(r.stdout, "Tmux session: %s\n", sessionRecord.TmuxSessionName)
	fmt.Fprintf(r.stdout, "Tmux window: %s\n", sessionRecord.TmuxWindow)
	fmt.Fprintf(r.stdout, "Tmux pane: %s\n", sessionRecord.TmuxPane)

	return nil
}

func (r Runner) executeAttach(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session identifier is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("attach does not accept extra positional arguments in the current milestone")
	}

	sessionRecord, err := r.loadSessionByIdentifier(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}
	if strings.TrimSpace(sessionRecord.TmuxSessionName) == "" || strings.TrimSpace(sessionRecord.TmuxPane) == "" {
		return fmt.Errorf("session %q does not have a live tmux binding", sessionRecord.ID)
	}

	fmt.Fprintf(r.stdout, "Attaching to %s (%s)\n", sessionRecord.ID, sessionRecord.TmuxPane)
	return r.app.Tmux.AttachPane(sessionRecord.TmuxSessionName, sessionRecord.TmuxPane)
}

func (r Runner) executeCapture(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session identifier is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("capture does not accept extra positional arguments in the current milestone")
	}

	sessionRecord, err := r.loadSessionByIdentifier(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}
	if strings.TrimSpace(sessionRecord.TmuxPane) == "" {
		return fmt.Errorf("session %q does not have a live tmux pane binding", sessionRecord.ID)
	}

	output, err := r.app.Tmux.CapturePane(sessionRecord.TmuxPane)
	if err != nil {
		return err
	}

	fmt.Fprint(r.stdout, output)
	return nil
}

type taskView struct {
	Task  task.Record
	Steps []step.Record
}

func (r Runner) printProjectSummary(title string, result *project.OpenResult, workspace *tmux.Workspace, sessions []session.Record, taskCount int, taskViews []taskView) {
	terminalAvailability := r.app.Tmux.Availability()
	workspaceName := r.app.Tmux.ProjectSessionName(result.SessionPrefix)

	fmt.Fprintln(r.stdout, title)
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintf(r.stdout, "Name: %s\n", result.Project.Name)
	fmt.Fprintf(r.stdout, "Repo: %s\n", result.Project.RepoPath)
	fmt.Fprintf(r.stdout, "Default branch: %s\n", result.Project.DefaultBranch)
	fmt.Fprintf(r.stdout, "DB: %s\n", result.DBPath)
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Terminal:")
	fmt.Fprintf(r.stdout, "  Driver: %s\n", result.TerminalDriver)
	fmt.Fprintf(r.stdout, "  Available: %t\n", terminalAvailability.Available)
	if terminalAvailability.Available {
		fmt.Fprintf(r.stdout, "  Binary: %s\n", terminalAvailability.BinaryPath)
	} else {
		fmt.Fprintln(r.stdout, "  Binary: not found")
	}
	fmt.Fprintf(r.stdout, "  Workspace: %s\n", workspaceName)
	if workspace != nil {
		state := "reused"
		if workspace.Created {
			state = "created"
		}
		fmt.Fprintf(r.stdout, "  Workspace state: %s\n", state)
	}
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Agents:")
	for _, agent := range result.Agents {
		fmt.Fprintf(r.stdout, "  - %s | role=%s | runtime=%s | enabled=%t\n", agent.Name, agent.Role, agent.Runtime, agent.Enabled)
	}
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Sessions:")
	if len(sessions) == 0 {
		fmt.Fprintln(r.stdout, "  None")
	} else {
		for _, item := range sessions {
			fmt.Fprintf(
				r.stdout,
				"  - %s | agent=%s | role=%s | runtime=%s | status=%s | tmux=%s %s %s\n",
				item.ID,
				item.AgentName,
				item.RoleName,
				item.Runtime,
				item.Status,
				item.TmuxSessionName,
				item.TmuxWindow,
				item.TmuxPane,
			)
		}
	}
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Tasks:")
	if len(taskViews) == 0 {
		fmt.Fprintln(r.stdout, "  None")
	} else {
		for _, item := range taskViews {
			fmt.Fprintf(
				r.stdout,
				"  - %s | title=%s | mode=%s | status=%s | role=%s | agent=%s | steps=%d\n",
				item.Task.ID,
				item.Task.Title,
				item.Task.Mode,
				item.Task.Status,
				emptyFallback(item.Task.PreferredRole),
				emptyFallback(item.Task.PreferredAgent),
				len(item.Steps),
			)
			fmt.Fprintf(r.stdout, "    next=%s\n", recommendTaskAction(item.Task.Status, item.Steps))
			for _, taskStep := range item.Steps {
				fmt.Fprintf(
					r.stdout,
					"    * %s | type=%s | title=%s | status=%s | role=%s | agent=%s | dependencies=%s\n",
					taskStep.ID,
					taskStep.StepType,
					taskStep.Title,
					taskStep.Status,
					emptyFallback(taskStep.RoleName),
					emptyFallback(taskStep.AgentName),
					formatDependencies(taskStep.Dependencies),
				)
			}
		}
	}
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Counts:")
	fmt.Fprintf(r.stdout, "  Tasks: %d\n", taskCount)
	fmt.Fprintf(r.stdout, "  Sessions: %d\n", len(sessions))
}

func (r Runner) printHelp() {
	fmt.Fprintln(r.stdout, "AOM")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Milestone 3 workflow scaffolding is in progress.")
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "Planned commands:")
	fmt.Fprintln(r.stdout, "  aom project init")
	fmt.Fprintln(r.stdout, "  aom attach")
	fmt.Fprintln(r.stdout, "  aom capture")
	fmt.Fprintln(r.stdout, "  aom open")
	fmt.Fprintln(r.stdout, "  aom plan")
	fmt.Fprintln(r.stdout, "  aom step list")
	fmt.Fprintln(r.stdout, "  aom step update")
	fmt.Fprintln(r.stdout, "  aom session show")
	fmt.Fprintln(r.stdout, "  aom session spawn")
	fmt.Fprintln(r.stdout, "  aom session list")
	fmt.Fprintln(r.stdout, "  aom status")
	fmt.Fprintln(r.stdout, "  aom task close")
	fmt.Fprintln(r.stdout, "  aom task create")
	fmt.Fprintln(r.stdout, "  aom task show")
	fmt.Fprintln(r.stdout, "  aom task update")
}

func findAgent(agents []agent.Record, name string) (*agent.Record, error) {
	for _, item := range agents {
		if item.Name != name {
			continue
		}
		if !item.Enabled {
			return nil, fmt.Errorf("agent %q is disabled", name)
		}

		agentCopy := item
		return &agentCopy, nil
	}

	return nil, fmt.Errorf("agent %q not found", name)
}

func sessionLaunchCommand(record session.Record, mockRuntime bool) string {
	if mockRuntime {
		return mockRuntimeShellCommand(record)
	}

	return placeholderShellCommand(record)
}

func placeholderShellCommand(record session.Record) string {
	return fmt.Sprintf(
		"sh -lc 'printf \"AOM session %s\\nagent=%s\\nrole=%s\\nruntime=%s\\n\"; exec ${SHELL:-sh}'",
		record.ID,
		record.AgentName,
		record.RoleName,
		record.Runtime,
	)
}

func mockRuntimeShellCommand(record session.Record) string {
	return fmt.Sprintf(
		"sh -lc 'printf \"AOM mock runtime boot\\nsession=%s\\nagent=%s\\nrole=%s\\nruntime=%s\\nstate=ready-for-operator\\n\"; exec ${SHELL:-sh}'",
		record.ID,
		record.AgentName,
		record.RoleName,
		record.Runtime,
	)
}

func launchModeLabel(mockRuntime bool) string {
	if mockRuntime {
		return "mock"
	}

	return "placeholder"
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}

	return value
}

func formatDependencies(values []string) string {
	if len(values) == 0 {
		return "-"
	}

	return strings.Join(values, ",")
}

func buildPlanStepSeeds(steps []plan.StepProposal) []task.StepSeed {
	if len(steps) == 0 {
		return nil
	}

	seeds := make([]task.StepSeed, 0, len(steps))
	for _, item := range steps {
		seed := task.StepSeed{
			Type:      item.Type,
			Title:     item.Title,
			RoleName:  item.RoleName,
			AgentName: item.AgentName,
		}
		seeds = append(seeds, seed)
	}
	return seeds
}

func recommendTaskAction(status string, steps []step.Record) string {
	if status == "Done" {
		return "task is closed; archive later if needed"
	}
	if status == "NeedsAttention" {
		return "operator review is needed before work continues"
	}
	for _, item := range steps {
		if item.Status == "NeedsAttention" {
			return "resolve the step that needs operator attention"
		}
	}
	for _, item := range steps {
		if item.Status == "Blocked" {
			return "unblock the blocked step or move it to NeedsAttention"
		}
	}
	if status == "Planned" && len(steps) > 0 && steps[0].Status == "Proposed" {
		return "confirm the proposed step and move the task to Ready"
	}
	for _, item := range steps {
		if item.Status == "Ready" {
			return fmt.Sprintf("start step %s", item.ID)
		}
		if item.Status == "InProgress" {
			return fmt.Sprintf("continue step %s", item.ID)
		}
	}

	return "inspect steps and choose the next operator action"
}

func (r Runner) loadTaskViews(result *project.OpenResult) ([]taskView, error) {
	taskService, taskDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer taskDB.Close()

	tasks, err := taskService.ListByProject(result.Project.ID)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, nil
	}

	stepService, stepDB, err := r.app.OpenStepService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer stepDB.Close()

	views := make([]taskView, 0, len(tasks))
	for _, item := range tasks {
		steps, err := stepService.ListByTask(item.ID)
		if err != nil {
			return nil, err
		}
		views = append(views, taskView{
			Task:  item,
			Steps: steps,
		})
	}

	return views, nil
}

func (r Runner) syncTaskArtifacts(result *project.OpenResult, taskID string, event artifact.Event, seed bool) error {
	return r.syncTaskArtifactsWithSession(result, taskID, event, seed, nil)
}

func (r Runner) syncTaskArtifactsWithSession(result *project.OpenResult, taskID string, event artifact.Event, seed bool, activeSession *session.Record) error {
	view, err := r.loadTaskView(result, taskID)
	if err != nil {
		return err
	}
	if view == nil {
		return fmt.Errorf("task %q not found for artifact sync", taskID)
	}

	service := artifact.NewService(result.Project.RepoPath, result.StateDir)
	params := artifact.SyncParams{
		Task:                  view.Task,
		Steps:                 view.Steps,
		ActiveSession:         activeSession,
		CreatedBy:             "operator",
		UpdatedBy:             event.Actor,
		RecommendedNextAction: recommendTaskAction(view.Task.Status, view.Steps),
	}
	if seed {
		return service.SeedTaskArtifacts(params)
	}
	if err := service.RefreshTaskArtifacts(params); err != nil {
		return err
	}
	return service.AppendEvent(taskID, event)
}

func (r Runner) loadTaskByID(result *project.OpenResult, taskID string) (*task.Record, error) {
	taskService, taskDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer taskDB.Close()

	return taskService.Get(taskID)
}

func (r Runner) loadStepByID(result *project.OpenResult, stepID string) (*step.Record, error) {
	stepService, stepDB, err := r.app.OpenStepService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer stepDB.Close()

	return stepService.Get(stepID)
}

func (r Runner) loadTaskView(result *project.OpenResult, taskID string) (*taskView, error) {
	taskService, taskDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer taskDB.Close()

	record, err := taskService.Get(taskID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}

	stepService, stepDB, err := r.app.OpenStepService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer stepDB.Close()

	steps, err := stepService.ListByTask(record.ID)
	if err != nil {
		return nil, err
	}

	return &taskView{
		Task:  *record,
		Steps: steps,
	}, nil
}

func mapTaskEventType(status, mode string) string {
	if strings.TrimSpace(mode) != "" {
		return "task.mode_changed"
	}
	if strings.EqualFold(strings.TrimSpace(status), "done") {
		return "task.closed"
	}
	return "task.updated"
}

func mapStepEventType(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "confirmed":
		return "step.confirmed"
	case "completed":
		return "step.completed"
	default:
		return "step.updated"
	}
}

func (r Runner) loadSessionByIdentifier(identifier string) (*session.Record, error) {
	result, err := r.app.Projects.Open(".")
	if err != nil {
		return nil, err
	}

	sessionService, sqlDB, err := r.app.OpenSessionService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	record, err := sessionService.Get(identifier)
	if err != nil {
		return nil, err
	}
	if record != nil {
		return record, nil
	}

	sessions, err := sessionService.ListByProject(result.Project.ID)
	if err != nil {
		return nil, err
	}

	for _, item := range sessions {
		if item.AgentName == identifier {
			sessionCopy := item
			return &sessionCopy, nil
		}
	}

	return nil, fmt.Errorf("session %q not found", identifier)
}

func (r Runner) loadProjectSessions(result *project.OpenResult) ([]session.Record, error) {
	sessionService, sqlDB, err := r.app.OpenSessionService(result.DBPath)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	return sessionService.ListByProject(result.Project.ID)
}

func (r Runner) loadTaskCount(result *project.OpenResult) (int, error) {
	taskService, sqlDB, err := r.app.OpenTaskService(result.DBPath)
	if err != nil {
		return 0, err
	}
	defer sqlDB.Close()

	return taskService.CountByProject(result.Project.ID)
}
