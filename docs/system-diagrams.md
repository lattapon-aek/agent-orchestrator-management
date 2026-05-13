# AOM System Diagrams

Visual reference for system architecture, state machines, and key flows.

---

## 1. System Architecture

```mermaid
graph TD
    subgraph Operator["Operator Layer"]
        Human["Human / AI Orchestrator Session"]
    end

    subgraph CLI["internal/cli — thin command layer"]
        Root["root.go\ncommand dispatch"]
    end

    subgraph App["internal/app — dependency wiring"]
        AppCore["app.go\nNew()"]
    end

    subgraph Domain["Domain Services"]
        Plan["internal/plan\nmode inference"]
        Project["internal/project\nconfig + agents"]
        Task["internal/task\nstate transitions"]
        Step["internal/step\nstep lifecycle"]
        Session["internal/session\nsession lifecycle"]
        Worktree["internal/worktree\ngit mapping"]
        Artifact["internal/artifact\n.agent/*.md writes"]
        Runtime["internal/runtime\nlaunch command builder"]
    end

    subgraph Infra["Infrastructure"]
        DB["internal/db\nSQLite bootstrap"]
        Tmux["internal/tmux\ntmux manager"]
        Config["internal/config\nYAML loader"]
    end

    subgraph External["External Systems"]
        SQLite[(".aom/sessions.db")]
        TmuxProc["tmux process"]
        AgentFiles[(".agent/*.md\nartifacts")]
        AgentProc["Agent Process\ncodex / claude / kiro"]
    end

    Human --> Root
    Root --> AppCore
    AppCore --> Plan
    AppCore --> Project
    AppCore --> Task
    AppCore --> Step
    AppCore --> Session
    AppCore --> Worktree
    AppCore --> Artifact
    AppCore --> Runtime
    AppCore --> Tmux
    Project --> Config
    Task --> DB
    Step --> DB
    Session --> DB
    Worktree --> DB
    DB --> SQLite
    Artifact --> AgentFiles
    Tmux --> TmuxProc
    TmuxProc --> AgentProc
    Runtime --> TmuxProc
```

---

## 2. Package Dependency Direction

```mermaid
graph LR
    cmd["cmd/aom/main.go"]
    cli["internal/cli"]
    app["internal/app"]
    domain["internal/{project,agent\ntask,step,session\nworktree,artifact,plan}"]
    infra["internal/{config,db,tmux}"]
    runtime["internal/runtime"]

    cmd --> cli
    cli --> app
    app --> domain
    app --> infra
    app --> runtime
    domain --> infra
```

---

## 3. Three-Layer Truth Model

```mermaid
graph LR
    C["③ Live tmux panes\nephemeral — replaceable"]
    B["② SQLite DB\n.aom/sessions.db\nstructured queries"]
    A["① .agent/*.md artifacts\ndurable — authoritative\ntask.md · state.md\nindex.md · log.md"]

    C -- "pane lost → reconcile\nSession → Detached" --> B
    B -- "conflict → defer to\nartifacts win" --> A

    style A fill:#2d6a4f,color:#fff,stroke:#1b4332
    style B fill:#1d3557,color:#fff,stroke:#0d1b2a
    style C fill:#6c757d,color:#fff,stroke:#495057
```

---

## 4. Task State Machine

```mermaid
stateDiagram-v2
    [*] --> Draft : task create

    Draft --> Planned : mode + steps confirmed
    Draft --> Archived

    Planned --> Ready : operator confirms
    Planned --> NeedsAttention

    Ready --> InProgress : session working
    Ready --> Archived

    InProgress --> Blocked : known blocker
    InProgress --> NeedsAttention : operator decision needed
    InProgress --> Done : operator closes
    InProgress --> Ready : step reset

    Blocked --> Ready : blocker resolved
    Blocked --> NeedsAttention

    NeedsAttention --> Planned
    NeedsAttention --> Ready
    NeedsAttention --> InProgress
    NeedsAttention --> Done

    Done --> Archived
    Archived --> [*]

    note right of NeedsAttention
        Real state gate.
        Requires operator decision
        before work continues.
    end note
```

---

## 5. Session State Machine

```mermaid
stateDiagram-v2
    [*] --> Created : session spawn

    Created --> Booting : pane launching
    Created --> Archived

    Booting --> Idle : pane ready
    Booting --> Detached : pane lost during boot
    Booting --> Failed : boot failed

    Idle --> Working : agent active
    Idle --> Detached : pane lost
    Idle --> Stopped : operator stop

    Working --> Idle : agent paused
    Working --> WaitingApproval : approval needed
    Working --> WaitingHandoff : work segment done
    Working --> Blocked : known blocker
    Working --> Detached : pane lost
    Working --> Failed : continuity broken

    WaitingApproval --> Working : approved
    WaitingApproval --> Blocked
    WaitingApproval --> Detached
    WaitingApproval --> Failed

    WaitingHandoff --> Idle : ready for next task
    WaitingHandoff --> Detached
    WaitingHandoff --> Stopped

    Blocked --> Idle
    Blocked --> Detached
    Blocked --> Failed

    Detached --> Idle : pane restored
    Detached --> Failed : continuity broken
    Detached --> Stopped

    Failed --> Archived
    Stopped --> Archived
    Archived --> [*]

    note right of WaitingHandoff
        Session may be reused
        for next task via
        aom session send
        (preserves agent context)
    end note
```

---

## 6. Worktree State Machine

```mermaid
stateDiagram-v2
    [*] --> Planned : task create

    Planned --> Provisioning : git worktree add
    Planned --> Archived

    Provisioning --> Ready : worktree created
    Provisioning --> NeedsRepair : git error

    Ready --> Active : session spawned on task
    Ready --> NeedsRepair : path drift detected
    Ready --> Archived

    Active --> Ready : session stopped / detached
    Active --> NeedsRepair : path drift detected

    NeedsRepair --> Ready : aom worktree repair
    NeedsRepair --> Archived

    Archived --> [*]

    note right of NeedsRepair
        DriftMissingPath
        DriftUnregisteredArtifactOnlyPath
        DriftUnregisteredDirtyPath
    end note
```

---

## 7. Step State Machine

```mermaid
stateDiagram-v2
    [*] --> Proposed : step seeded

    Proposed --> Confirmed : operator confirms
    Proposed --> Skipped
    Proposed --> Canceled

    Confirmed --> Ready : role + agent assigned
    Confirmed --> Canceled

    Ready --> InProgress : session working
    Ready --> Skipped
    Ready --> Canceled

    InProgress --> Blocked : known blocker
    InProgress --> NeedsAttention : operator decision needed
    InProgress --> Completed : step contract met
    InProgress --> Ready : reset

    Blocked --> Ready
    Blocked --> NeedsAttention

    NeedsAttention --> Ready
    NeedsAttention --> InProgress
    NeedsAttention --> Canceled

    Completed --> [*]
    Skipped --> [*]
    Canceled --> [*]
```

---

## 8. Session Spawn Flow

```mermaid
sequenceDiagram
    participant Op as Operator / Orchestrator
    participant CLI as aom CLI
    participant DB as SQLite DB
    participant WT as Git Worktree
    participant Tmux as tmux
    participant Art as .agent/*.md

    Op->>CLI: aom session spawn <agent> --task <id> --real
    CLI->>DB: project.Open() — load config + agents
    CLI->>DB: worktree.Reconcile() — verify worktree health
    CLI->>CLI: runtime.Builder.Build() — validate binary in PATH
    Note over CLI: fail here if runtime unsupported<br/>before any pane is created
    CLI->>DB: session.Create() — persist record (Booting)
    CLI->>Art: artifact.AppendEvent(session.created)
    CLI->>Tmux: tmux.CreatePane(worktreePath, launchCmd)
    Tmux->>WT: cd <worktree> && exec codex/claude
    CLI->>DB: session.Save() — persist pane binding (Idle)
    CLI->>DB: worktree.Reconcile(hasActiveSession=true) → Active
    CLI->>Art: artifact.AppendEvent(session.ready)
    CLI->>Tmux: tmux.AnnotatePane(@aom_session_id, @aom_agent)
    CLI-->>Op: Session spawned — pane ready
```

---

## 9. AI Orchestrator Loop

```mermaid
sequenceDiagram
    participant O as Claude Code\n(Orchestrator)
    participant AOM as aom CLI
    participant Art as .agent/*.md
    participant Agent as Sub-Agent\n(codex / claude)

    O->>AOM: aom plan "feature X" --create
    AOM->>Art: seed task.md, state.md, index.md, log.md
    AOM->>AOM: git worktree add → Worktree Ready
    AOM-->>O: Task + Worktree created

    O->>AOM: aom session spawn backend-main --task <id> --real
    AOM->>Agent: launch in tmux pane (exec codex)
    AOM-->>O: Session spawned (Idle)

    O->>AOM: aom session send <id> "read .agent/task.md and begin"
    AOM->>Agent: tmux send-keys → agent reads task.md

    loop Agent working
        Agent->>Art: update state.md (progress)
    end

    Agent->>Art: write handoff.md (completed work)
    Agent->>Art: append handoff.prepared to log.md

    O->>Art: poll log.md → detect handoff.prepared
    O->>Art: read handoff.md (20-30 lines)
    Note over O: context stays small —\norchestrator reads summaries\nnot terminal output

    alt work complete
        O->>AOM: aom task update --status done
    else needs review
        O->>AOM: aom session spawn reviewer-main --task <id> --real
        O->>AOM: aom session send <reviewer> "read .agent/handoff.md and review"
    else loop back
        O->>AOM: aom session send <id> "next task: read updated task.md"
        Note over O,Agent: same session reused —\nagent context preserved
    end
```

---

## 10. Artifact Lifecycle

```mermaid
graph TD
    TC["task create / plan --create"]
    SS["session spawn --task"]
    SU["step update"]
    TU["task update"]
    AI["aom attach\n(operator.intervention)"]
    SR["session replace"]
    WR["worktree repair"]
    CP["checkpoint"]
    HO["handoff"]

    TC -->|"seed"| TM["task.md"]
    TC -->|"seed"| SM["state.md"]
    TC -->|"seed"| IM["index.md"]
    TC -->|"seed"| LM["log.md"]
    TC -->|"seed if mode"| RM["requirements.md\ndesign.md\ntasks.md"]

    SS -->|"refresh + append session.created\nsession.ready"| IM
    SS -->|"append events"| LM

    SU -->|"refresh + append step.updated"| IM
    SU -->|"append"| LM

    TU -->|"refresh + append task.updated"| SM
    TU -->|"refresh"| IM
    TU -->|"append"| LM

    AI -->|"refresh + append operator.intervention"| IM
    AI -->|"append"| LM

    SR -->|"refresh + append session.replaced"| IM
    SR -->|"append"| LM

    WR -->|"append worktree.repaired"| LM

    CP -->|"append checkpoint.created"| LM
    CP -->|"refresh"| IM

    HO -->|"write / refresh"| HM["handoff.md"]
    HO -->|"append handoff.prepared"| LM
    HO -->|"refresh"| IM

    subgraph "AOM-owned (never written by agents)"
        IM
        LM
    end

    subgraph "Agent-updated under AOM protocol"
        TM
        SM
        HM
        RM
    end
```

---

## 11. Operator Definition (Human vs AI)

```mermaid
graph TD
    subgraph Current["Current Milestone (Human Operator)"]
        H["Human\nproject owner"]
        H -->|"runs aom CLI directly"| AOM1["aom commands"]
        AOM1 --> Agents1["sub-agent sessions"]
    end

    subgraph Future["AI Orchestrator Model"]
        HO["Human\nproject owner\n(override authority)"]
        O["Claude Code\norchestrator session\nrole: orchestrator\nruntime: claude"]
        HO -->|"monitors + overrides"| O
        O -->|"runs aom CLI commands"| AOM2["aom commands"]
        AOM2 --> Agents2["sub-agent sessions\ncodex / claude / kiro"]
    end

    note1["All state transitions remain\nexplicit CLI commands\nregardless of who drives them"]
```

---

## 12. Runtime Identity File Delivery

```mermaid
graph TD
    subgraph ProjectLevel[".aom/ (project-level, AOM-owned)"]
        P1[".aom/agents/backend-codex/profile.md"]
        P2[".aom/agents/backend-claude/profile.md"]
        P3[".aom/agents/reviewer-main/profile.md"]
        P4[".aom/agents/orchestrator-main/profile.md"]
    end

    subgraph SpawnTime["session spawn (materialization)"]
        AOM["AOM reads profile.md\nfor the named agent"]
        W1["codex runtime\n→ writes AGENTS.md\nin worktree root"]
        W2["claude runtime\n→ writes CLAUDE.md\nin worktree root"]
        W3["gemini runtime\n→ writes GEMINI.md\nin worktree root"]
    end

    subgraph WorktreeRuntime["Worktree (runtime CWD)"]
        WT[".aom/worktrees/TASK-001-slug/\n  AGENTS.md  ← found at ./\n  CLAUDE.md\n  .agent/task.md\n  .agent/state.md"]
    end

    P1 --> AOM
    P2 --> AOM
    P3 --> AOM
    P4 --> AOM
    AOM --> W1
    AOM --> W2
    AOM --> W3
    W1 --> WT
    W2 --> WT
    W3 --> WT

    note["Profile in .aom/agents/ = authoritative source\nWorktree copy = spawn-time materialization\nRuntime discovers it at ./ via normal traversal"]
```

---

## 13. Multi-Session Agent Model

```mermaid
graph TD
    subgraph AgentDef["Agent Definition (project-level)"]
        A["backend-codex\nruntime: codex\nrole: backend"]
    end

    subgraph Task1["TASK-001 worktree"]
        S1["Session A\nWorking"]
        WT1[".aom/worktrees/TASK-001/\n  AGENTS.md\n  .agent/task.md"]
    end

    subgraph Task2["TASK-002 worktree"]
        S2["Session B\nWorking"]
        WT2[".aom/worktrees/TASK-002/\n  AGENTS.md\n  .agent/task.md"]
    end

    A -->|"spawn session A"| S1
    A -->|"spawn session B"| S2
    S1 --> WT1
    S2 --> WT2

    note1["Same agent definition\nTwo isolated sessions\nEach has own worktree + identity copy\nNo conflict"]

    style note1 fill:#2d6a4f,color:#fff,stroke:#1b4332
```
