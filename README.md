# AOM — Agents Orchestrator Management

A CLI control plane for managing multiple AI agent sessions (Claude Code, Codex) as a coordinated team. One operator runs `aom` to dispatch tasks, manage agent sessions, and maintain durable state across git worktrees.

---

## Build

```bash
go build -o aom cmd/aom/main.go
```

### Building on WSL2 / Linux

The `aom` binary in this repo is a macOS executable. WSL2 and Linux users must build from source:

```bash
# Install Go 1.24+ if not present
# https://go.dev/dl/

export GOTOOLCHAIN=local
export GOCACHE=$PWD/.cache/gocache
export GOMODCACHE=$PWD/.cache/gomodcache
export GOTELEMETRY=off

go build -o aom cmd/aom/main.go
./aom doctor   # verify environment
```

> **NTFS worktree note:** If your repo lives on an NTFS mount (e.g. `/mnt/c/...`), `git commit` inside
> worktrees may fail with `index.lock: Read-only file system`. Use `aom worktree commit <task-id>`
> as a drop-in replacement — it resolves the lock issue automatically.

---

## Quick Start

### 1. Initialize a project

```bash
cd /your/repo
aom project init my-project --repo .
aom doctor                 # verify environment
```

This creates `.aom/` with `project.yaml`, `agents.yaml`, `policy.yaml`, `resources.yaml`, and `sessions.db`.

### 2. Plan and create tasks

```bash
# Preview a plan (no side effects)
aom plan "build a login system"

# Create and persist a task from a plan
aom plan "build a login system" --mode requirements-first --create

# Or create tasks manually
aom task create "Implement auth API" --role backend --agent codex-main --priority high
aom task create "Build login page"   --role frontend --agent claude-main --priority high
aom task create "Security review"    --role reviewer --agent reviewer-main
```

### 3. Wire dependencies

```bash
aom task link T2 --blocked-by T1    # T2 waits for T1
aom task link T3 --blocked-by T1
aom task link T4 --blocked-by T2
aom task link T4 --blocked-by T3

aom next                            # see which tasks are ready to start
aom status                          # full project overview
```

### 4. Spawn agent sessions

```bash
# Real agents (requires claude / codex in PATH)
aom session spawn claude-main --task T3

# Mock mode — no real agent, useful for testing
aom session spawn claude-main --task T3 --mock
aom session spawn codex-main  --task T2 --mock
```

### 5. Communicate with agents

```bash
# Send a prompt to a session
aom session send SESS-xxx "Start with requirements.md"

# Broadcast to multiple sessions
aom broadcast "Daily standup in 5 min" --sessions SESS-1,SESS-2

# Team channel (shared log all agents can read)
aom channel append "T1 complete. API spec: POST /login, /logout, /me" --agent claude-main
aom channel read

# P2P mailbox between agents
aom message send codex-main "Use httpOnly cookie, not JWT"
aom message read codex-main
```

### 6. Track progress

```bash
aom status                          # full task + session overview
aom task list                       # tabular task list
aom session list                    # all sessions
aom session show SESS-xxx           # single session detail
aom capture SESS-xxx                # snapshot current pane output
```

### 7. Task lifecycle

```bash
# Advance steps (required before task close)
aom step list T2 --ids-only         # get step IDs
aom step update STEP-xxx --status completed

# Close a task (marks Done, warns if uncommitted git work)
aom task close T2

# Merge task branch into main
aom merge check   T2               # check for file overlaps
aom merge prepare T2 --into main   # write merge-plan.md
aom merge commit  T2 --into main   # git merge --no-ff (requires commits on branch)
```

### 8. Session recovery

```bash
aom checkpoint SESS-xxx             # save progress snapshot
aom handoff SESS-xxx --to codex-main  # pass work to another agent
aom session stop SESS-xxx
aom session replace SESS-xxx --agent claude-main --reason "timeout" --mock
aom session resume SESS-xxx --task T4
aom session archive SESS-xxx
```

---

## Common Workflows

### Pause and resume all work

```bash
aom pause-all --reason "deploying hotfix"
# ... do other work ...
aom resume-all
```

### CI feedback loop

```bash
aom task record-result T4 --failed --summary "3 tests failing on /api/login"
# agent fixes it
aom task record-result T4 --passed --summary "all 12 tests pass"
```

### Review workflow

```bash
aom review T5 --mock               # spawn review session
aom review close T5                # close after fixes applied
```

### Cross-worktree inspection

```bash
aom worktree read-file T2 server.js      # read file from T2's worktree
aom worktree repair T2                   # fix broken worktree
```

### Observability

```bash
aom metrics                         # team velocity report
aom team brief                      # full team state snapshot
aom session health --all            # time since last checkpoint per session
aom watch --task T4 --timeout 5m   # stream log events live
```

---

## agents.yaml — Team Configuration

```yaml
roles:
  frontend:
    class: builder
    worktree_mode: dedicated-writer
    checkpoint_expectation: required

  backend:
    class: builder
    worktree_mode: dedicated-writer
    checkpoint_expectation: required

  reviewer:
    class: reviewer
    worktree_mode: read-only
    checkpoint_expectation: optional

agents:
  claude-main:
    runtime: claude
    role: frontend
    enabled: true

  codex-main:
    runtime: codex
    role: backend
    enabled: true

  reviewer-main:
    runtime: claude
    role: reviewer
    enabled: true
```

---

## Testing with `--mock`

All session commands accept `--mock` to run without real agents:

```bash
aom session spawn claude-main --task T1 --mock
```

Mock sessions boot a shell pane with session metadata printed but no AI agent running. All state transitions, artifacts, and git operations work normally — only the AI process is absent.

For a full end-to-end smoke test:

```bash
bash scripts/e2e-smoke.sh          # 43 checks, all --mock, no agents required
```

---

## Key Files After Init

```
.aom/
  project.yaml        project name, repo path, default branch
  agents.yaml         agent definitions and role configs
  policy.yaml         deny_commands, approval settings
  resources.yaml      skill files, MCP servers, role bindings
  sessions.db         SQLite — all task/session/step/worktree state
  channel.md          shared team message log
  mailbox/<agent>.md  per-agent P2P inbox
  project-board.md    auto-updated task status board
  team-brief.md       latest team state snapshot
  worktrees/<task>/   one git worktree per task
    .agent/
      task.md         task brief for the agent
      state.md        current step and status
      index.md        continuity readiness summary
      log.md          canonical event log (source of truth)
      handoff.md      filled by agent before handoff
```

---

## Further Reading

| Document | Covers |
|---|---|
| `docs/current-status.md` | Implementation progress and handoff notes |
| `docs/AOM-planning.md` | Full vision and product goals |
| `docs/state-machine.md` | Task, session, step, worktree state machines |
| `docs/artifact-schemas.md` | `.agent/*.md` file schemas |
| `docs/cli-spec.md` | Full CLI command reference |
| `CLAUDE.md` | Guidelines for AI implementation partners |
