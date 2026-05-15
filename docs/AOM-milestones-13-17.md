# AOM Milestone Plan: M13–M17

## Purpose

This document extends the original milestone plan with the next five milestones.
Milestones 1–12 established the operational plumbing: sessions, worktrees, handoffs,
governance, and channel broadcast. Milestones 13–17 add the workflow intelligence
layer — the features that make a group of agents behave like a coordinated team
rather than a manually-relayed pipeline.

---

## Milestone 13: Task Graph and Priority

### Goal

Let tasks declare dependencies on other tasks and carry a priority level, so the
orchestrator can answer "what should the team work on next?" without guessing.

### Context

Currently the orchestrator must manually track which tasks are blocked by others.
There is no first-class dependency model at the task level (step-level dependencies
within a single task already exist). There is also no priority field, so the operator
must remember urgency mentally.

### Scope

- cross-task dependency model (junction table in SQLite)
- cycle detection when adding dependencies
- priority field on tasks (high / normal / low)
- `aom task link` and `aom task unlink` commands
- `--priority` flag on `aom task create` and `aom task update`
- `aom next` command: ordered list of unblocked tasks by priority
- blocked-by and priority surfaced in `index.md` and `aom status`
- `task.linked` and `task.unlinked` log events

### Deliverables

- schema-v5 migration: `task_dependencies` junction table, `priority` column on `tasks`
- dependency CRUD methods in `internal/task/repository.go`
- cycle detection (BFS) and priority normalization in `internal/task/service.go`
- `BlockedBy`, `Unblocks`, `ListUnblocked` service methods
- updated `index.md` rendering with blocked-by list and priority
- `aom task link <task-id> --blocks <other-task-id>`
- `aom task unlink <task-id> --blocks <other-task-id>`
- `aom next` command

### Suggested commands

```
aom task link   <task-id> --blocks <other-task-id>
aom task unlink <task-id> --blocks <other-task-id>
aom task create "<title>" --priority <high|normal|low>
aom task update <task-id> --priority <high|normal|low>
aom next
```

`aom next` output: unblocked tasks ordered by priority (high first), then creation
order. Blocked tasks shown separately with `waiting on: TASK-xxx` annotation.

### Acceptance criteria

- a task can be linked as blocked by another task
- `aom next` lists unblocked tasks with high-priority tasks first
- blocked tasks do not appear in `aom next` until their blockers are `Done`
- adding a dependency that would create a cycle is rejected with an error
- `index.md` shows the blocked-by list and priority field
- `aom status` shows a dependency indicator next to blocked tasks

### Risks addressed

- orchestrator must manually remember task ordering
- no machine-readable priority for automated orchestration loops
- accidental circular dependencies in multi-agent task graphs

---

## Milestone 14: Agent Self-Service and Team Briefing

### Goal

Agents can signal that a new subtask is needed without the operator manually creating
it; new agents joining a mid-progress project get a single structured brief instead of
reading every artifact individually.

### Context

Currently only the operator can create tasks. In practice, agents discover subtasks
during implementation (a missing API endpoint, an undocumented constraint) and can only
signal this via freeform channel messages. There is also no structured onboarding
artifact for a new agent joining an active project.

### Scope

**Part A — Agent-initiated subtask requests**

- agents write requests to `.aom/requests/` via `aom task request`
- requests are pending artifacts awaiting operator approval
- operator approves or rejects with `aom task approve-request` / `aom task reject-request`
- approved requests create real tasks using the existing task creation flow
- `aom status` shows a pending requests count section
- `aom task list-requests` shows all pending requests

**Part B — Team briefing artifact**

- `aom team brief` generates `.aom/team-brief.md`
- brief includes: active task table (with priority and blocked-by), pending requests,
  last 5 channel messages, agent table with session status
- `session spawn --task` prints the `team-brief.md` path alongside `task.md` so
  the agent gets full team context on boot

### Deliverables

- `.aom/requests/<id>.md` artifact schema and write/read helpers
- `aom task request`, `aom task list-requests`, `aom task approve-request`,
  `aom task reject-request` commands
- `request.created`, `request.approved`, `request.rejected` log events
- `GenerateTeamBrief()` in `internal/artifact/service.go`
- `aom team brief` command
- `session spawn` output updated to include team-brief path

### Suggested commands

```
aom task request "<title>" [--from-session <session-id>] [--priority <level>]
aom task list-requests
aom task approve-request <request-id>
aom task reject-request  <request-id> [--reason "<why>"]
aom team brief
```

### Request artifact schema

`.aom/requests/<id>.md`:

```
# Task Request: <title>
- ID: REQ-<timestamp>
- Requested by: <session-id> / <agent-name>
- Parent task: <task-id>
- Priority: <normal|high|low>
- Status: pending|approved|rejected
- Reason: <optional free text>
```

### Acceptance criteria

- an agent can file a request and the operator sees it in `aom status`
- approving a request creates a real task linked to the parent task
- rejecting a request records a reason and removes it from the pending list
- `aom team brief` produces a complete `.aom/team-brief.md`
- `session spawn --task` mentions the team-brief path in its output

### Risks addressed

- agents can only signal needs via unstructured channel messages
- new agents must manually read multiple artifacts to understand team state
- orchestrator context grows large when relaying agent discovery back into task creation

---

## Milestone 15: Merge Coordination

### Goal

When two agents finish parallel tasks on separate branches, AOM surfaces the overlap
risk before the operator manually investigates which files conflict.

### Context

Multi-agent parallel work creates diverging branches. The operator currently has no
tool to detect file-level overlap between two active task branches before attempting a
merge. Discovering conflicts at merge time is late and disruptive.

### Scope

- `aom merge check` dry-run overlap analysis (reads git, writes nothing)
- conflict scoring: Green (0 overlaps), Yellow (1–3), Red (>3)
- `aom merge prepare` creates a `merge-plan.md` artifact and an integration step
- new `"integration"` step type in the step model
- `merge-plan.md` artifact schema

### Deliverables

- `internal/merge/` package: `DetectOverlaps()`, `ScoreConflicts()`
- `merge-plan.md` artifact template in `internal/artifact/service.go`
- `"integration"` added to allowed step types in `internal/step/repository.go`
- `aom merge check` and `aom merge prepare` commands
- `merge-plan.md` schema documented in `docs/artifact-schemas.md`

### Suggested commands

```
aom merge check   <task-id> [--against <other-task-id|branch>]
aom merge prepare <task-id> [--into <branch>]
```

`aom merge check` output:

```
Merge check: TASK-xxx → main
Conflict score: Yellow (2 overlapping files)

Overlapping files:
  internal/auth/token.go   also modified in TASK-yyy (agent: backend-main)
  internal/api/router.go   also modified in TASK-yyy (agent: backend-main)

Recommended: review overlapping files with TASK-yyy owner before merging.
```

### merge-plan.md schema

```markdown
# Merge Plan
- Task: TASK-xxx
- Target branch: main
- Prepared at: <timestamp>
- Conflict score: Green|Yellow|Red

## File Overlaps
- `path/to/file.go` — also modified in TASK-yyy (agent: backend-main)

## Recommended actions
- [ ] Review overlapping files with TASK-yyy owner
- [ ] Run tests after merge
```

### Acceptance criteria

- `aom merge check` reports overlapping files between two task branches without
  modifying any file or branch
- conflict score reflects the number of overlapping files
- `aom merge prepare` writes `merge-plan.md` and creates an integration step owned
  by the operator role
- the integration step type is accepted by the step state machine

### Risks addressed

- file conflicts discovered only at merge time after significant rework
- no structured handoff artifact for the integration phase of parallel work
- operator must manually run git commands to understand branch divergence

---

## Milestone 16: Communication and Feedback Upgrade

### Goal

Close the feedback loops that the current system leaves open: direct agent-to-agent
messaging, automated test result ingestion, context health warnings, and a bulk
pause/resume mechanism for operator-controlled holds.

### Context

The current channel (`aom channel`) is broadcast-only with no privacy model. There is
no way for one agent to send a message specifically to another. CI results are not
tracked. The operator has no signal when an agent's context is growing stale. Stopping
all agents for a review requires stopping each session individually.

### Scope

**Part A — P2P agent messaging (mailboxes)**

- per-agent mailbox files at `.aom/mailbox/<agent-name>.md`
- `aom message send`, `aom message read`, `aom message clear`
- `session spawn --task` prints unread message count for the spawning agent

**Part B — CI/test feedback loop (passive)**

- `aom task record-result` receives pass/fail from external CI pipelines
- failed result transitions task to `NeedsAttention`
- result stored in `state.md` and appended to `log.md`
- AOM does not trigger CI; CI pipelines call AOM after finishing

**Part C — Context health monitoring**

- `aom session health <session-id>` reports time since last checkpoint
- warns when > 2 hours since last checkpoint or no handoff after 4 hours
- `aom session health --all` shows a summary table for all active sessions
- health warnings appear in `aom status` output next to session rows

**Part D — Emergency pause and resume**

- `aom pause-all [--reason "<why>"]` transitions all `Working` sessions to
  `WaitingApproval` and broadcasts a pause message to each
- `aom resume-all` bulk-approves all sessions in `WaitingApproval`
- both commands append canonical log events to each affected task

### Deliverables

- `internal/cli/message.go` (new): mailbox append/read/clear helpers
- `aom message send`, `aom message read`, `aom message clear` commands
- `.aom/mailbox/<agent-name>.md` artifact format
- `aom task record-result` command
- `test.passed` and `test.failed` log event types
- `HealthReport()` method in `internal/session/service.go`
- `aom session health` command (single and `--all`)
- `aom pause-all` and `aom resume-all` commands

### Suggested commands

```
aom message send <agent-name> "<message>" [--from <sender>]
aom message read [--agent <name>]
aom message clear <agent-name>

aom task record-result <task-id> --passed | --failed [--summary "<text>"] [--url <ci-url>]

aom session health <session-id>
aom session health --all

aom pause-all  [--reason "<why>"]
aom resume-all
```

### Mailbox format

`.aom/mailbox/<agent-name>.md`:

```markdown
# Mailbox: <agent-name>

## Messages

### <timestamp> | MSG-<id> | from: <sender>
<message text>
```

### Acceptance criteria

- `aom message send backend-main "..."` appends to `.aom/mailbox/backend-main.md`
- `aom message read --agent backend-main` prints that mailbox
- `aom task record-result TASK-xxx --failed` moves the task to `NeedsAttention`
- `aom session health --all` shows a table with time-since-checkpoint per session
- `aom pause-all` transitions every `Working` session to `WaitingApproval`
- `aom resume-all` approves all sessions in `WaitingApproval`

### Risks addressed

- agents cannot ask each other questions without broadcasting to the full channel
- CI failures are not visible in AOM task state without manual `task update`
- operator has no signal when an agent's context is getting stale
- stopping all agents for a team review requires running multiple stop commands

---

## Milestone 17: Observability

### Goal

Give the operator and orchestrator tools to understand what is happening across the
team without reading individual artifact files: cross-worktree file access and a
team velocity dashboard.

### Context

Gemini and Kiro runtime support is deferred until CLI flags are confirmed and live
E2E testing is available. M17 focuses on the observability gap: an agent in one
worktree cannot read a file from another agent's worktree, and there is no aggregate
view of how the team is performing over time.

### Scope

**Part A — Cross-worktree read access**

- `aom worktree read-file <task-id> <relative-path>` reads a file from any task's
  worktree in read-only mode
- path traversal prevention using `filepath.Clean` and prefix check
- audit trail: `worktree.read` event appended to the requesting task's log

**Part B — Team velocity metrics**

- `aom metrics [--days 7] [--task <id>]` derives stats from existing log events and
  task DB records (no new tables needed)
- summary table: tasks completed, avg duration, tasks blocked > 1 hour
- per-agent table: tasks owned, completed, avg time to completion
- bottleneck hint: agent with most block events surfaced explicitly

### Deliverables

- `aom worktree read-file` command in `internal/cli/root.go`
- worktree path validation helper (no shell exec, pure `os.ReadFile`)
- `worktree.read` log event type
- `internal/cli/metrics.go` (new)
- `VelocityStats()` method in `internal/task/service.go`
- `aom metrics` command

### Suggested commands

```
aom worktree read-file <task-id> <relative-path>
aom metrics [--days <n>] [--task <id>]
```

### Acceptance criteria

- `aom worktree read-file TASK-xxx src/main.go` prints the file contents without
  modifying the worktree
- a path like `../../etc/passwd` is rejected with an error
- `aom metrics --days 7` prints a summary of completed tasks and agent utilization
- `aom metrics` output includes a bottleneck hint when one agent has significantly
  more block events than others

### Risks addressed

- agents must go through a full handoff to access a single file from another task
- operator has no aggregate view of team throughput or where bottlenecks accumulate

---

## Implementation Order

| Milestone | Depends on | Why this order |
|-----------|------------|----------------|
| M13 — Task Graph & Priority | M1–M12 complete | Foundation; M14+ reference dependency state and `aom next` |
| M14 — Agent Self-Service & Brief | M13 | Team brief includes priority and blocked-by from M13 |
| M15 — Merge Coordination | M13 | Integration steps benefit from stable task graph |
| M16 — Communication & Feedback | M13–M14 | Independent additions; delay until workflow layer is solid |
| M17 — Observability | M13–M16 | Last-mile polish; velocity metrics need completed task history |

---

## Files Touched Per Milestone

### M13

| File | Change |
|------|--------|
| `internal/db/db.go` | Add schema-v5 migration |
| `internal/task/repository.go` | Add `Priority` field; add dependency CRUD |
| `internal/task/service.go` | Priority normalization; cycle detection; `BlockedBy`, `Unblocks`, `ListUnblocked` |
| `internal/artifact/service.go` | Render blocked-by and priority in `index.md` |
| `internal/cli/root.go` | Add `task link`, `task unlink`, `next`; `--priority` on create/update |
| `docs/artifact-schemas.md` | Document priority and blocked-by fields in `index.md` |

### M14

| File | Change |
|------|--------|
| `internal/artifact/service.go` | `WriteRequestArtifact`, `ReadPendingRequests`, `GenerateTeamBrief` |
| `internal/task/service.go` | `CreateFromRequest` |
| `internal/cli/root.go` | Add `task request`, `task list-requests`, `task approve-request`, `task reject-request`, `team brief` |
| `docs/artifact-schemas.md` | Document `request.md` and `team-brief.md` schemas |

### M15

| File | Change |
|------|--------|
| `internal/merge/` (new package) | `DetectOverlaps`, `ScoreConflicts` |
| `internal/artifact/service.go` | `WriteMergePlan` |
| `internal/step/repository.go` | Allow `"integration"` step type |
| `internal/cli/root.go` | Add `merge check`, `merge prepare` |
| `docs/artifact-schemas.md` | Document `merge-plan.md` schema |

### M16

| File | Change |
|------|--------|
| `internal/cli/message.go` (new) | Mailbox append / read / clear |
| `internal/cli/root.go` | Add `message`, `task record-result`, `session health`, `pause-all`, `resume-all` |
| `internal/session/service.go` | `HealthReport` method |
| `internal/artifact/service.go` | Append `test.passed` / `test.failed` events; update `state.md` test result field |

### M17

| File | Change |
|------|--------|
| `internal/cli/root.go` | Add `worktree read-file` |
| `internal/cli/metrics.go` (new) | `aom metrics` output |
| `internal/task/service.go` | `VelocityStats` method |
