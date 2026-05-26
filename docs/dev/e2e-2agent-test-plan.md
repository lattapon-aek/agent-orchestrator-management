# E2E Test Plan: 2-Agent Handoff (Builder → Reviewer)
> Phase 2 verification — พิสูจน์ว่า pipeline ครบวงจรไม่เสียเงียบๆ

**วันที่**: 2026-05-26  
**เป้า**: builder commit + signal → verify gate ผ่าน → reviewer spawn → review report → accept ทั้งคู่  
**Environment**: WSL2 Ubuntu, runtime = claude (haiku หรือ sonnet)

---

## Setup

```bash
# ใน WSL
export PATH="/tmp/aom-e2e-2agent:/usr/bin:/usr/local/bin:/bin:$PATH"
mkdir -p /tmp/aom-e2e-2agent

# build binary ล่าสุด (รันจาก Windows side ก่อน)
# GOOS=linux GOARCH=amd64 go build -o /tmp/aom-e2e-2agent/aom cmd/aom/main.go

# สร้างโปรเจคทดสอบ
mkdir -p /tmp/e2e-2agent && cd /tmp/e2e-2agent
git init && git config user.email "test@aom.local" && git config user.name "Test"
echo "# E2E 2-Agent Test" > README.md
git add README.md && git commit -m "init"

aom project init --name "e2e-2agent" --default-branch main
```

---

## Phase A: Provision agents

```bash
# เพิ่ม 2 agents
aom agent add backend-1 --role backend --class builder --runtime claude
aom agent add reviewer-1 --role reviewer --class reviewer --runtime claude

# provision workspaces
aom agent provision backend-1
aom agent provision reviewer-1

# ตรวจสอบ
aom agent list
# ควรเห็น: backend-1 (workspace: .aom/agents/backend-1/workspace)
#          reviewer-1 (workspace: .aom/agents/reviewer-1/workspace)
```

---

## Phase B: สร้าง task และ steps

```bash
# สร้าง task สำหรับ builder
TASK_ID=$(aom task create "Create a hello world HTTP server in Python" \
  --agent backend-1 --format json | jq -r '.id')
echo "Task: $TASK_ID"

# เพิ่ม step
STEP_ID=$(aom step add $TASK_ID "Write server.py with GET /hello endpoint" \
  --format json | jq -r '.id')
echo "Step: $STEP_ID"

aom task ready $TASK_ID
```

---

## Phase C: Spawn builder และรอให้เสร็จ

```bash
# spawn builder session
SESSION_ID=$(aom session spawn backend-1 --task $TASK_ID --real --format json | jq -r '.id')
echo "Session: $SESSION_ID"

# ส่งคำสั่งเริ่มต้น
aom session send $SESSION_ID "read .agent/task.md and begin implementing the task"

# monitor (รัน loop สั้นๆ หรือดูด้วยมือ)
watch -n 10 "aom status"

# ตรวจสอบ channel ว่า builder โพสต์อะไรบ้าง
aom channel read
```

**สิ่งที่ builder ต้องทำ (ตาม profile):**
1. อ่าน task.md + state.md + log.md
2. เขียน server.py + tests
3. อัปเดต state.md
4. `git add -A && git commit -m "[TASK-xxx] implement GET /hello"`
5. เขียน handoff.md
6. append `task.completed` ลง `.agent/log.md`
7. `aom step update <step-id> --status completed`
8. `aom channel append "backend-1: task done"`

---

## Phase D: Verify และรับงาน builder

```bash
# รัน verify ตรวจสอบ
aom task verify $TASK_ID

# ควรเห็น:
# [PASS] commits on branch
# [PASS] state.md updated
# [PASS] handoff.md filled
# [PASS] task.completed in log

# ถ้าผ่านทุก check
aom task accept $TASK_ID

# ตรวจสอบ handoff.md ที่ builder เขียน
cat .aom/tasks/$TASK_ID/handoff.md
```

---

## Phase E: Spawn reviewer

```bash
# สร้าง review task (linked กับ builder task)
REV_TASK_ID=$(aom task create "Review backend-1 implementation of hello server" \
  --agent reviewer-1 --format json | jq -r '.id')

# link dependency
aom task link $REV_TASK_ID --depends-on $TASK_ID

# spawn reviewer
REV_SESSION=$(aom session spawn reviewer-1 --task $REV_TASK_ID --real --format json | jq -r '.id')

# ส่ง context
aom session send $REV_SESSION "read .agent/task.md and review the implementation from backend-1"

# monitor
aom channel read
```

**สิ่งที่ reviewer ต้องทำ (ตาม profile):**
1. ตรวจสอบว่า branch มี commits จริง (`git log main..agents/backend-1`)
2. อ่าน handoff.md จาก builder
3. อ่านไฟล์จาก workspace ของ builder: `aom worktree read-file $TASK_ID server.py`
4. เขียน review-report.md
5. append `task.completed` ลง `.agent/log.md`
6. `aom channel append "reviewer-1: review done"`

---

## Phase F: Accept reviewer และ merge

```bash
# verify reviewer
aom task verify $REV_TASK_ID

# accept
aom task accept $REV_TASK_ID

# ดู review report
cat .aom/tasks/$REV_TASK_ID/review-notes.md  # หรือ review-report.md

# merge builder work
aom merge check $TASK_ID
aom merge prepare $TASK_ID
aom merge commit $TASK_ID

# ตรวจสอบว่า commit ขึ้นมาบน main
git log --oneline main | head -5
```

---

## Checklist ที่ต้องผ่านทั้งหมด

```
Setup
[ ] aom project init สำเร็จ
[ ] provision สร้าง .aom/agents/*/workspace/ จริง
[ ] git worktree list แสดงทั้งสอง workspace

Builder workflow
[ ] session spawn ไม่ error
[ ] builder เขียน server.py จริงใน workspace
[ ] commit มี prefix [TASK-xxx]
[ ] state.md ไม่มีบรรทัด "None recorded yet"
[ ] handoff.md ยาวกว่า 80 chars ไม่มี sentinel text
[ ] .agent/log.md มี "task.completed"
[ ] aom channel มีข้อความจาก backend-1

Verify gate
[ ] aom task verify $TASK_ID → ทุก check [PASS]
[ ] aom task accept $TASK_ID สำเร็จ (ไม่ต้องใช้ --force)

Reviewer workflow
[ ] reviewer อ่าน handoff.md ของ builder ได้
[ ] reviewer เขียน review findings
[ ] .agent/log.md ของ reviewer มี "task.completed"
[ ] aom task verify $REV_TASK_ID → ทุก check [PASS]
[ ] aom task accept $REV_TASK_ID สำเร็จ

Merge
[ ] aom merge check ไม่มี conflict
[ ] aom merge commit สำเร็จ
[ ] git log main แสดง commit จาก builder จริง
```

---

## สัญญาณที่บอกว่า Phase 2 สำเร็จ

1. Checklist ด้านบนผ่านครบโดยไม่ต้องใช้ `--force` แม้แต่ครั้งเดียว
2. Operator ไม่ต้องแก้ไขอะไรระหว่าง builder → reviewer handoff
3. `git log main` แสดง commit ของ builder ที่มี `[TASK-xxx]` prefix จริง
4. ไม่มี silent failure — ทุก error มี message บอกชัดว่าต้องทำอะไร

---

## Expected failure modes (และวิธีแก้)

| Failure | สาเหตุที่น่าจะเป็น | วิธีแก้ |
|---------|------------------|--------|
| verify: "task.completed not found" | agent ไม่ได้เขียน log.md เพราะสับสน F1 | ✅ F1 แก้แล้ว — ถ้ายังเกิด ดู profile |
| verify: "no commits on branch" | agent ไม่ใส่ `[TASK-xxx]` prefix | profile checklist บอกไว้ใน task.md |
| verify: "handoff.md too sparse" | agent เขียนสั้นเกินไป | ดู handoff.md แล้ว `aom task accept --force` ถ้าเนื้อหาพอ |
| reviewer: branch empty error | builder ยังไม่ commit | ดู reviewer profile — มี readiness check |
| merge: empty commit set | commits ไม่มี `[TASK-xxx]` | F2 fix ยังรอ — verify จะจับได้ถ้า implement |
