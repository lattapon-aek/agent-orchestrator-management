# AOM Use Cases

This document catalogs validated and anticipated use cases for AOM. Each case describes the problem, the team composition, the key AOM commands used, and what makes AOM's model better than running agents individually.

---

## Contents

1. [Full-Stack Feature Pipeline](#1-full-stack-feature-pipeline)
2. [Bug Investigation and Fix](#2-bug-investigation-and-fix)
3. [API-First Development](#3-api-first-development)
4. [Security Audit Pipeline](#4-security-audit-pipeline)
5. [Parallel Module Refactoring](#5-parallel-module-refactoring)
6. [Test Coverage Sprint](#6-test-coverage-sprint)
7. [Documentation Generation](#7-documentation-generation)
8. [Data Migration Pipeline](#8-data-migration-pipeline)
9. [Review-Fix Loop](#9-review-fix-loop)
10. [Multi-Service Feature Rollout](#10-multi-service-feature-rollout)

---

## 1. Full-Stack Feature Pipeline

**Status:** Validated — run on login-app project (2026-05-18)

### Problem

Building a full-stack feature (backend API + frontend UI + security review) typically requires context-switching between domains, careful handoff between developers, and a sequential workflow that wastes time waiting for dependencies.

### What AOM Enables

- Backend and frontend agents work in parallel once the API contract is defined
- Reviewer agent inspects both layers after implementation without re-reading all context
- Each agent stays isolated in its own git worktree — no accidental overwrites
- Orchestrator manages transitions with explicit CLI commands, not manual coordination

### Team Composition

```yaml
agents:
  backend-codex:
    runtime: codex    # o4-mini for Go backend
    role: backend

  frontend-claude:
    runtime: claude   # sonnet for Next.js frontend
    role: frontend

  reviewer-claude:
    runtime: claude
    role: reviewer
```

### Pipeline

```
T1 (Schema Design)
  └─ T2 (Go Auth API — backend-codex)
  └─ T3 (Frontend Scaffold — frontend-claude)  ← parallel with T2
       └─ T4 (Frontend Pages — frontend-claude)
            └─ T5 (Security Review — reviewer-claude)
```

### Key Commands

```bash
# Define the pipeline
aom task create "Design DB schema" --role backend --priority high
aom task create "Go auth API" --role backend --agent backend-codex --priority high
aom task create "Frontend scaffold" --role frontend --agent frontend-claude
aom task create "Frontend pages" --role frontend --agent frontend-claude
aom task create "Security review" --role reviewer --agent reviewer-claude

# Wire dependencies
aom task link T2 --depends-on T1
aom task link T3 --depends-on T1
aom task link T4 --depends-on T3
aom task link T5 --depends-on T2
aom task link T5 --depends-on T4

# See what's ready
aom next

# Spawn agents (T2 and T3 are parallel)
aom session spawn backend-codex --task T2 --real
aom session spawn frontend-claude --task T3 --real

# Wait for T2 and T3 to finish, then T4 starts
aom session wait <sess-T2> --event task.completed
aom session wait <sess-T3> --event task.completed

# AOM auto-emits task.unblocked to T4 when T3 closes
aom task close T3
aom session spawn frontend-claude --task T4 --real

# After T4 done, spawn reviewer
aom task close T4
aom session spawn reviewer-claude --task T5 --real

# Merge and close
aom merge check T2
aom merge commit T2 --into main
aom merge commit T4 --into main
```

### AOM Advantage

- Worktree isolation prevents backend and frontend changes from conflicting
- `task.unblocked` events remove the need for bash polling loops
- Reviewer gets a clean cross-worktree view of both completed layers
- Full audit trail in `log.md` across all agents

### Lessons Learned (from live run)

- Define the API contract (OpenAPI spec) before spawning backend and frontend in parallel — prevents integration guesswork
- `aom session wait --event task.completed` eliminates hand-written bash polling
- Brief agents with specific file paths, not natural language descriptions — agents produce on-spec output when the brief is precise
- Reviewer as a late-stage role catches bugs that context-saturated implementers miss

---

## 2. Bug Investigation and Fix

**Status:** Anticipated

### Problem

A production bug needs root-cause analysis before a fix can be written. Investigation and implementation are different cognitive modes — one benefits from wide reading, the other from focused writing.

### What AOM Enables

- A `Bugfix` mode task generates a structured investigation plan automatically
- Investigation step runs first; implementation runs only after root cause is confirmed
- If the bug spans multiple layers, hand off to a specialist for the fix

### Team Composition

```yaml
agents:
  investigator-claude:
    runtime: claude
    role: builder    # reads logs, traces, tests

  fixer-codex:
    runtime: codex
    role: backend    # implements the fix
```

### Pipeline

```
T1 (Investigate: aom plan "fix login timeout" --mode bugfix)
  └─ T2 (Implement fix — fixer-codex)
       └─ T3 (Regression review — investigator-claude)
```

### Key Commands

```bash
# Bugfix mode generates investigation + implementation steps
aom plan "fix login session timeout after 5 minutes" --mode bugfix --create

# Spawn investigator
aom session spawn investigator-claude --task T1 --real
aom session send <sess-T1> "@.agent/task.md — start with the investigation step"

# Wait for root cause to be documented
aom session wait <sess-T1> --event handoff.prepared

# Hand off to fixer with context from investigation
aom handoff <sess-T1> --to fixer-codex
aom session spawn fixer-codex --task T1 --real

# After fix, run regression review
aom task close T1
aom review T1
```

### AOM Advantage

- `Bugfix` mode plan includes: symptom capture → reproduction steps → root cause → fix → regression check
- Handoff carries investigation context to the fixer without re-reading logs
- One task ID tracks the full bug lifecycle from report to close

---

## 3. API-First Development

**Status:** Anticipated

### Problem

Teams often build frontend and backend in parallel based on a verbal API spec, leading to integration failures discovered late. API-first development requires a contract phase before parallel implementation.

### What AOM Enables

- T1 generates the OpenAPI spec as a shared artifact
- T2 (backend) and T3 (frontend) both read the spec from T1's worktree via `aom worktree read-file`
- No guessing — both agents work from the same machine-readable source of truth

### Pipeline

```
T1 (OpenAPI Spec Design)
  ├─ T2 (Backend implementation)   ← reads T1's openapi.yaml
  └─ T3 (Frontend API client)      ← reads T1's openapi.yaml
       └─ T4 (Integration tests)
```

### Key Commands

```bash
# After T1 is done and spec is committed
aom worktree read-file T1 api/openapi.yaml  # read from T1's worktree

# Brief T2 and T3 to read the spec
aom session send <sess-T2> "Start by reading the API contract: aom worktree read-file T1 api/openapi.yaml"
aom session send <sess-T3> "Start by reading the API contract: aom worktree read-file T1 api/openapi.yaml"

# Both can run parallel
aom session spawn backend-claude --task T2 --real
aom session spawn frontend-claude --task T3 --real
```

### AOM Advantage

- `aom worktree read-file` gives agents cross-worktree read access without copying files
- Spec changes in T1 are visible to T2/T3 on every read — no stale copies
- Integration failures are caught before merge because T4 runs against merged output

---

## 4. Security Audit Pipeline

**Status:** Anticipated

### Problem

Security audits require multiple passes: static analysis, manual code review, dependency scanning, and a consolidated report. These are sequential concerns but each is time-consuming.

### Pipeline

```
T1 (Static analysis + dependency scan — automated agent)
T2 (Manual code review — security-claude)               ← reads T1 findings
T3 (Penetration test checklist — security-claude)
T4 (Consolidated report + remediation plan — orchestrator)
```

### Key Commands

```bash
aom task create "Static analysis" --role security --priority high
aom task create "Manual code review" --role security --priority high
aom task create "Pentest checklist" --role security
aom task create "Security report" --role orchestrator

aom task link T2 --depends-on T1
aom task link T4 --depends-on T2
aom task link T4 --depends-on T3

# Agent reads T1's findings from its worktree
aom session send <sess-T2> "Read T1's findings: aom worktree read-file T1 findings.md"

# After T2 and T3 close, AOM auto-notifies T4 via task.unblocked
aom session wait <sess-T4> --event task.unblocked
```

### AOM Advantage

- Findings accumulate in artifact files (`review-notes.md`) that survive session restarts
- `aom task record-result` tracks which findings are resolved vs open
- `aom metrics` shows time spent per audit phase for future estimation

---

## 5. Parallel Module Refactoring

**Status:** Anticipated

### Problem

A large codebase needs refactoring across 3–5 independent modules. Sequential refactoring takes weeks; parallel refactoring risks merge conflicts.

### What AOM Enables

- Each module gets its own task and git worktree — agents never touch each other's files
- `aom merge check` detects overlap before any merge
- `aom next --format json` lets the orchestrator dispatch agents as modules become available

### Team Composition

```yaml
agents:
  refactor-1: { runtime: claude, role: builder }
  refactor-2: { runtime: claude, role: builder }
  refactor-3: { runtime: codex,  role: builder }
```

### Pipeline

```
T1 (Module: auth)    T2 (Module: payments)    T3 (Module: notifications)
     ↓                     ↓                          ↓
T4 (Integration test — all three merged to a staging branch)
```

### Key Commands

```bash
# Check for overlap before merging any module
aom merge check T1 --against T2
aom merge check T1 --against T3

# Merge in sequence once clean
aom merge commit T1 --into staging
aom merge commit T2 --into staging
aom merge commit T3 --into staging

# Integration test
aom session spawn refactor-1 --task T4 --real
```

### AOM Advantage

- Git worktree isolation eliminates race conditions during parallel editing
- `aom merge check` surfaces file-level overlaps before any destructive operation
- Agents work at full throughput across all modules simultaneously

---

## 6. Test Coverage Sprint

**Status:** Anticipated

### Problem

A codebase has 40% test coverage and needs to reach 80%. Writing tests for all packages simultaneously is the fastest path but risks agents conflicting on shared test infrastructure.

### Pipeline

```
T1 (Test: internal/auth)      T2 (Test: internal/payments)     T3 (Test: internal/api)
          ↓                               ↓                              ↓
T4 (Coverage report + gap analysis)
```

### Key Commands

```bash
# Spawn 3 agents in parallel, each owns one package
aom session spawn claude-1 --task T1 --real
aom session spawn claude-2 --task T2 --real
aom session spawn codex-1  --task T3 --real

# Each agent reports coverage on completion
# Task record-result captures the metric
aom task record-result T1 --passed --summary "auth: 94% coverage"
aom task record-result T2 --passed --summary "payments: 87% coverage"

# When all three close, T4 unblocks automatically (task.unblocked event)
aom session wait <sess-T4> --event task.unblocked
```

### AOM Advantage

- No shared mutable state between test files in different packages
- `aom task record-result` stores CI metrics alongside the task for future reference
- `aom metrics --days 1` shows which agent finished fastest → inform future assignment

---

## 7. Documentation Generation

**Status:** Anticipated

### Problem

Generating documentation (API reference, architecture guide, user guide) requires reading code, understanding it, and writing in different voices. These are parallel workstreams.

### Pipeline

```
T1 (API reference — reads codebase)
T2 (Architecture guide — reads system diagrams + source)
T3 (User guide — reads product requirements)
     ↓
T4 (Docs review + consistency pass)
```

### Key Commands

```bash
# Docs agents read source from the main repo worktree
aom session send <sess-T1> "Read the source and generate docs. Key files: aom worktree read-file MAIN internal/api/handlers.go"

# Cross-agent consistency: T4 reads all three draft docs
aom session send <sess-T4> "Review consistency across: aom worktree read-file T1 docs/api.md / T2 docs/arch.md / T3 docs/guide.md"

# Channel for coordination if T1 finds something T2 needs to know
aom channel append "API rate-limit behavior changed in auth handler — see T1 worktree" --agent claude-1
```

### AOM Advantage

- `aom worktree read-file` lets the review agent read all three draft docs without copying
- Channel messages allow discoverable async coordination between doc agents
- Each doc agent works in its own branch — no merge conflicts on different doc files

---

## 8. Data Migration Pipeline

**Status:** Anticipated

### Problem

A database migration involves: schema design → migration script → data validation → rollback script. Each step must complete before the next can start, and failure at any step must be caught before affecting production.

### Pipeline

```
T1 (Schema design + review)
  └─ T2 (Migration script + dry-run)
       └─ T3 (Data validation queries)
            └─ T4 (Rollback script + runbook)
```

### Key Commands

```bash
# Requirements-first mode for schema design
aom plan "migrate user table to add soft-delete" --mode requirements-first --create

# Each step waits on the previous via task.unblocked
aom task link T2 --depends-on T1
aom task link T3 --depends-on T2
aom task link T4 --depends-on T3

# Pause all agents before production migration window
aom pause-all --reason "production migration window starting"
# ... run migration in prod ...
aom resume-all

# Record migration result
aom task record-result T3 --passed --summary "validated 2.4M rows, 0 errors"
```

### AOM Advantage

- Sequential dependency chain prevents running validation before migration is written
- `aom pause-all` / `aom resume-all` coordinates all agents around a maintenance window
- `aom task record-result` captures the outcome for post-mortem reference

---

## 9. Review-Fix Loop

**Status:** Anticipated

### Problem

Code review often takes multiple iterations: reviewer finds issues → implementer fixes → reviewer re-checks. Manual handoff between these roles is slow and loses context.

### What AOM Enables

- Reviewer writes findings to `review-notes.md` — structured, persistent, counts unresolved items
- `aom review close` transitions task back to implementer automatically
- `aom handoff` carries reviewer findings to the fixing agent without re-reading the full context
- Loop repeats until `review-notes.md` shows 0 unresolved items

### Key Commands

```bash
# Spawn reviewer
aom review T1

# Reviewer writes findings to review-notes.md
# When done, reviewer signals handoff.prepared

# Check unresolved count before closing review
aom task show T1   # shows "Unresolved review items: 3"

# Hand back to implementer
aom handoff <reviewer-sess> --to backend-codex
aom review close T1   # transitions task back to InProgress

# Implementer fixes and signals task.completed
# Reviewer re-checks
aom review T1         # re-runs review on updated code
```

### AOM Advantage

- `review-notes.md` survives session restarts — reviewer never loses previous findings
- Unresolved item count surfaces in `aom status` and `aom task show` so operator can monitor progress
- When all findings point to one role, AOM auto-suggests the correct follow-up owner

---

## 10. Multi-Service Feature Rollout

**Status:** Anticipated

### Problem

A product feature spans multiple microservices (auth service, billing service, notification service). Each service needs its own agent with domain expertise, but they share a common API contract and need to ship together.

### What AOM Enables

- Each service is a separate task with its own worktree and agent
- A shared contract artifact (OpenAPI or protobuf) is written first and read cross-worktree
- Integration testing task unblocks only when all service tasks are done

### Team Composition

```yaml
agents:
  auth-agent:    { runtime: claude, role: backend }
  billing-agent: { runtime: codex,  role: backend }
  notif-agent:   { runtime: claude, role: backend }
  qa-agent:      { runtime: claude, role: reviewer }
```

### Pipeline

```
T0 (Contract definition — shared protobuf / OpenAPI)
  ├─ T1 (Auth service changes)
  ├─ T2 (Billing service changes)     ← all parallel after T0
  └─ T3 (Notification service changes)
       └─ T4 (Integration test + QA sign-off)
```

### Key Commands

```bash
# All service tasks depend on T0 (the contract)
aom task link T1 --depends-on T0
aom task link T2 --depends-on T0
aom task link T3 --depends-on T0
aom task link T4 --depends-on T1
aom task link T4 --depends-on T2
aom task link T4 --depends-on T3

# Agents read the shared contract
aom session send <sess-T1> "Read contract: aom worktree read-file T0 proto/api.proto"
aom session send <sess-T2> "Read contract: aom worktree read-file T0 proto/api.proto"
aom session send <sess-T3> "Read contract: aom worktree read-file T0 proto/api.proto"

# Each service agent coordinates via channel if needed
aom channel read   # see what other services have announced

# When T1, T2, T3 all close, T4 auto-unblocks
aom next --format json | jq '.unblocked[] | select(.id == "T4")'

# Merge all three services before integration test
aom merge check T1 --against T2
aom merge commit T1 --into staging
aom merge commit T2 --into staging
aom merge commit T3 --into staging
```

### AOM Advantage

- All three service agents work simultaneously with zero chance of file conflicts (separate worktrees)
- `task.unblocked` event on T4 fires automatically when T1 + T2 + T3 all close
- Channel allows loose coordination between services (announcing breaking changes, etc.)
- `aom merge check` prevents silent conflicts before staging merge

---

## When AOM Works Best

Based on the use cases above, AOM's model produces the most value when:

| Condition | Why |
|-----------|-----|
| Interface between components is defined upfront (schema, API spec, protobuf) | Agents can work in parallel without guessing about the contract |
| Each task maps to a clear domain (backend, frontend, reviewer) | Role specialization gives agents focused context |
| Tasks have clear completion criteria | Agents can signal `task.completed` reliably |
| Work spans multiple git-isolated modules | Worktree isolation prevents conflicts |
| Sequential gating is required (T2 can't start before T1) | `task link --depends-on` + `task.unblocked` events automate the gate |

## When AOM Needs More Setup

| Condition | What to add |
|-----------|-------------|
| Agents need to ask clarifying questions mid-task | Use `aom message send` + `aom message read` for async P2P |
| Design decisions change mid-pipeline | `aom pause-all`, revise brief, `aom resume-all` |
| Spec is ambiguous at start | Use `requirements-first` or `design-first` plan mode before spawning agents |
| Runtime is sandboxed (codex, WSL/NTFS) | Use outbox pattern; put worktrees on native filesystem |
