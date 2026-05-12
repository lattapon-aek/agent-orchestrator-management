# Current Status

## Purpose

This document is the current handoff point for the repository.

It should be enough for a developer or agent to:
- understand what is already implemented
- verify the current state quickly
- continue the next milestone without re-discovering context

## Current Milestone Status

### Milestone 0

Completed.

Foundation specs are in place:
- [AOM planning](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\AOM-planning.md)
- [Milestone plan](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\AOM-milestones.md)
- [State machine](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\state-machine.md)
- [Artifact schemas](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\artifact-schemas.md)
- [Project config](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\project-config.md)
- [CLI spec](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\cli-spec.md)
- [Project structure](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\project-structure.md)
- [Engineering guidelines](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\engineering-guidelines.md)

### Milestone 1

Completed.

Implemented:
- Go module bootstrap
- config loader and validation
- SQLite bootstrap and migrations
- `aom project init`
- `aom open`
- `aom status`

Main reference:
- [Milestone 1 plan](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\milestone-1-implementation-plan.md)

### Milestone 2

Completed in code, tests, and live local E2E on macOS.

Implemented:
- tmux manager skeleton
- tmux availability detection
- stable workspace naming
- session schema v2
- session repository and service
- workspace create or reuse on `aom open`
- `aom session spawn`
- `aom session list`
- `aom session show`
- `aom attach`
- `aom capture`
- session-aware `aom status`

Main reference:
- [Milestone 2 plan](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\milestone-2-implementation-plan.md)

### Milestone 3

Started.

Implemented in the first slice:
- task schema v3 additions
- step table
- task repository and service
- step repository
- `aom task create`
- `aom task show`
- `aom step list`
- task-aware `aom status` counts

Implemented in the second slice:
- `aom task update`
- `aom task close`
- `aom step update`
- task status transition validation
- step status transition validation
- step `Ready` owner validation

Implemented in the third slice:
- `aom plan`
- lightweight orchestrator recommendation service
- mode inference for `Direct`, `Bugfix`, `Requirements-first`, and `Design-first`
- proposed step generation without immediate task creation
- `aom plan --create` to persist accepted planning output into a task with seeded steps
- template-based project init bootstrap for baseline config files
- `aom project init --template` for preset starter templates
- `aom project init --template-dir` for external starter templates
- richer `aom status` task and step visibility with recommended next action hints

### Milestone 4

Started.

Implemented in the first slice:
- artifact generator in `internal/artifact`
- task creation seeds `task.md`, `state.md`, `index.md`, and `log.md`
- structured modes seed mode-specific artifacts such as `requirements.md`, `design.md`, and `tasks.md`
- task and step updates refresh task artifacts and append canonical log events
- pre-worktree canonical artifact root at `.aom/tasks/<task-id>/`
- `session spawn --task` refreshes task artifacts with active session context and appends `session.created`

## Current CLI Surface

Implemented commands:
- `aom project init`
- `aom open`
- `aom plan`
- `aom status`
- `aom task create`
- `aom task update`
- `aom task close`
- `aom task show`
- `aom step list`
- `aom step update`
- `aom session spawn`
- `aom session list`
- `aom session show`
- `aom attach`
- `aom capture`

Current behavior notes:
- `open` ensures tmux workspace and fails clearly when tmux is unavailable
- `plan` gives a lightweight orchestrator recommendation by default, and `plan --create` persists it into a task with seeded steps
- `project init` renders baseline config from template assets instead of hardcoded agent structs
- `project init --template` lets a project pick a preset starter team from `templates/project-init/<name>`
- `project init --template-dir` lets a project supply its own starter config templates
- `status` shows project, terminal summary, agents, sessions, detailed task rows, step summaries, and task-level recommended next action hints
- `task create` defaults to `Direct` mode and creates one initial `Proposed` implementation step
- `task create` and `plan --create` now seed task-local continuity artifacts under `.aom/tasks/<task-id>/`
- `task update` and `step update` validate allowed state transitions, including `NeedsAttention`
- `session spawn --task` binds `task_id` into the session record and refreshes `state.md`, `index.md`, and `log.md`
- `session spawn --mock` launches a mock runtime transcript for live local flow verification
- `session spawn` otherwise uses a placeholder shell command, not a real provider CLI yet
- `attach` and `capture` operate through the tmux manager abstraction

## Current Packages

### Working packages

- [cmd/aom/main.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\cmd\aom\main.go)
- [internal/app/app.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\app\app.go)
- [internal/app/sessions.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\app\sessions.go)
- [internal/cli/root.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\cli\root.go)
- [internal/config/config.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\config\config.go)
- [internal/db/db.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\db\db.go)
- [internal/project/service.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\project\service.go)
- [internal/project/repository.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\project\repository.go)
- [internal/artifact/service.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\artifact\service.go)
- [internal/project/templates/project-init/agents.yaml.tmpl](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\project\templates\project-init\agents.yaml.tmpl)
- [templates/project-init/default/agents.yaml.tmpl](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\templates\project-init\default\agents.yaml.tmpl)
- [templates/project-init/minimal/agents.yaml.tmpl](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\templates\project-init\minimal\agents.yaml.tmpl)
- [internal/plan/service.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\plan\service.go)
- [internal/agent/repository.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\agent\repository.go)
- [internal/session/repository.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\session\repository.go)
- [internal/session/service.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\session\service.go)
- [internal/step/repository.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\step\repository.go)
- [internal/task/repository.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\task\repository.go)
- [internal/task/service.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\task\service.go)
- [internal/tmux/manager.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\tmux\manager.go)

### Tests

- [internal/config/config_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\config\config_test.go)
- [internal/db/db_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\db\db_test.go)
- [internal/project/repository_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\project\repository_test.go)
- [internal/project/service_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\project\service_test.go)
- [internal/artifact/service_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\artifact\service_test.go)
- [internal/plan/service_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\plan\service_test.go)
- [internal/agent/repository_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\agent\repository_test.go)
- [internal/session/repository_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\session\repository_test.go)
- [internal/session/service_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\session\service_test.go)
- [internal/step/repository_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\step\repository_test.go)
- [internal/task/repository_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\task\repository_test.go)
- [internal/task/service_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\task\service_test.go)
- [internal/tmux/manager_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\tmux\manager_test.go)
- [internal/cli/root_test.go](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\internal\cli\root_test.go)

## Verified State

Last verified state before this handoff:
- `go test ./...` passes
- live local Milestone 2 flow passes on macOS:
  - `aom project init aom --repo .`
  - `aom open`
  - `aom status`
  - `aom session spawn backend-main`
  - `aom session list`
  - `aom session show <session-id>`
  - `aom capture <session-id>`
- live first-slice Milestone 3 flow passes on macOS:
  - `aom task create "Implement milestone 3 slice" --role backend --agent backend-main`
  - `aom task show <task-id>`
  - `aom step list <task-id>`
  - `aom status`
- live second-slice Milestone 3 flow passes on macOS:
  - `aom task update <task-id> --mode bugfix --status ready`
  - `aom step update <step-id> --status confirmed`
  - `aom step update <step-id> --status ready`
  - `aom task update <task-id> --status in-progress`
  - `aom task close <task-id>`
- live third-slice Milestone 3 flow passes on macOS:
  - `aom plan "fix login bug"`
  - `aom plan "fix checkout bug" --create`
  - `aom project init template-check --repo /private/tmp/aom-template-init-check --template-dir ./internal/project/templates/project-init`
  - `aom project init minimal-check --repo /private/tmp/aom-template-minimal-check --template minimal`
  - `aom status`
  - `aom session spawn reviewer-main --mock`
  - `aom capture SESS-1778508537180275000`
- live first-slice Milestone 4 flow passes on macOS:
  - `aom task create "Seed artifact layer" --role backend --agent backend-main`
  - inspect `.aom/tasks/TASK-1778509207359142000/{task,state,index,log}.md`
  - `aom plan "capture auth requirements" --mode requirements-first --create`
  - inspect `.aom/tasks/TASK-1778509234475738000/{task,state,index,log,requirements,tasks}.md`
  - `aom session spawn backend-main --task TASK-1778509474319106000 --mock`
  - inspect `.aom/tasks/TASK-1778509474319106000/{index,log}.md`

Suggested verification commands on a new machine:

```powershell
$env:GOTOOLCHAIN='local'
$env:GOCACHE="$PWD\.cache\gocache"
$env:GOMODCACHE="$PWD\.cache\gomodcache"
$env:GOTELEMETRY='off'
$env:GOTELEMETRYDIR="$PWD\.cache\gotelemetry"
& 'C:\Program Files\Go\bin\go.exe' test ./...
```

## Environment Notes

### Go

This repo has been verified with:
- Go 1.24.x

The repository currently uses:
- `gopkg.in/yaml.v3`
- `modernc.org/sqlite`

### tmux and Live E2E

Current state:
- live tmux E2E is verified on macOS with working `tmux`
- the earlier Windows environment did not successfully run live tmux E2E
- the earlier Windows execution context did not have a working `tmux` path or usable `wsl.exe`

What this means:
- code and tests for tmux logic pass
- live local tmux behavior is verified on macOS
- provider runtime launch is still placeholder-only and not yet provider-native E2E

Recommended path for live E2E:
- Linux or macOS should work best for continued live runtime validation
- Windows still needs a working WSL + tmux path if it is used again for live checks

## What Is Intentionally Not Done Yet

Still out of scope at the current handoff point:
- real provider runtime launch for Codex, Claude, or Kiro
- worktree-aware session spawn
- handoff and checkpoint logic
- provider-native resume and replacement flows
- worktree provisioning and moving artifact roots from repo fallback into real task worktrees

## Immediate Next Step

Next milestone to continue:
- `Milestone 4: Operational Memory Layer`

Recommended first implementation slice:
1. append richer canonical log events around session lifecycle
2. move artifact root from repo fallback to task worktree when Milestone 5 begins
3. start task-to-worktree mapping so task-bound sessions launch in isolated paths

## Suggested First Checks On Another Machine

1. Clone the repo and open the root directory.
2. Read:
   - [AGENTS.md](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\AGENTS.md)
   - [docs/project-structure.md](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\project-structure.md)
   - [docs/engineering-guidelines.md](C:\Users\lattapon.kea\Desktop\Agents-Orchestfator-Management\docs\engineering-guidelines.md)
   - this file
3. Run `go test ./...`
4. If tmux is available, manually test:
   - `aom project init`
   - `aom open`
   - `aom session spawn backend-main`
   - `aom session list`
   - `aom capture <session-id>`
5. If tmux is not available, continue with Milestone 3 and keep tmux E2E deferred.
