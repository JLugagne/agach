# Implementation Specification - Critical Clarifications

## IMPORTANT CHANGES FROM PROJECT.MD

### 1. Authentication & Multi-tenancy
- **NO authentication** - This is a single-instance system
- **NO Actor/UserID/TeamID/TenantID** - Remove all actor-related code
- No user management, no sessions, no auth middleware

### 2. Column Structure (CHANGED)
**4 FIXED COLUMNS:**
1. **Todo** (`todo`, position 0) - Tasks to be done
2. **In Progress** (`in_progress`, position 1, WIP limit 3) - Tasks being worked on
3. **Done** (`done`, position 2) - Completed tasks
4. **Blocked** (`blocked`, position 3) - Tasks requiring human intervention

**NO "Won't Do" column** - Tasks needing human review go to "Blocked"

### 3. Blocking Workflow (CHANGED)
- Agent encounters problem → **moves task to "blocked" column** (not stays in todo)
- Agent calls `block_task` → task moves to `blocked` column
- Blocked tasks are **visible in the Blocked column**, not flagged in Todo
- Only humans can unblock via web UI → moves back to `todo`

### 4. Won't-Do Workflow (CHANGED)
- Agent calls `request_wont_do` → task **moves to "blocked" column** with `wont_do_requested = 1`
- Human sees task in Blocked column with "Won't Do Requested" badge
- Human approves → task deleted or archived (discuss with user)
- Human rejects → adds comment, moves task back to `todo`, clears `wont_do_requested`

### 5. Priority System
- **Simple 4-level system ONLY**:
  - `critical` → 400
  - `high` → 300
  - `medium` → 200
  - `low` → 100
- **NO dynamic boosting**, **NO urgency calculations**, **NO dependency boosting**

### 6. Task Fields (ADDED)
**New required field: `summary`**
- Filled when task is created
- Brief description of what needs to be done
- Different from `description` (which is more detailed)

**New field: `resolution`**
- Filled when agent stops working on a task
- Describes work done so far (for incomplete work)
- When human moves task from "in progress" back to "todo", system appends:
  ```
  [Moved back to Todo by human - task was not completed]
  ```
- If field already has content, append to end (don't alter existing content)

### 7. Task Lifecycle

**Agent creates task:**
```
create_task(title, summary, description, ...) → task in "todo"
```

**Agent picks up task:**
```
get_next_task(project_id, role) → returns highest priority task
move_task(task_id, "in_progress") → moves to In Progress column
```

**Agent completes task:**
```
complete_task(task_id, completion_summary, files_modified) → moves to "done"
```

**Agent stops work (incomplete):**
```
update_task(task_id, resolution="Tried X, Y failed because Z. Next agent should...")
move_task(task_id, "todo") → back to backlog for another agent
```

**Agent encounters blocker:**
```
block_task(task_id, blocked_reason) → moves to "blocked" column
# Human sees it, adds comment with guidance, clicks "Unblock" → moves to "todo"
```

**Agent thinks task is unnecessary:**
```
request_wont_do(task_id, reason) → moves to "blocked" with wont_do_requested flag
# Human approves → delete task
# Human rejects → add comment, move to "todo", clear flag
```

**Human moves in-progress task back to todo:**
```
# Via UI drag-and-drop or "Move to Todo" button
# System automatically appends to resolution field:
resolution += "\n\n[Moved back to Todo by human on YYYY-MM-DD - task was not completed]"
```

### 8. Database Schema Changes

```sql
CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,          -- "todo", "in_progress", "done", "blocked"
    name TEXT NOT NULL,                 -- "To Do", "In Progress", "Done", "Blocked"
    position INTEGER NOT NULL,
    wip_limit INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    column_id TEXT NOT NULL REFERENCES columns(id),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,               -- NEW: Brief summary (required at creation)
    description TEXT DEFAULT '',
    priority TEXT DEFAULT 'medium' CHECK(priority IN ('critical','high','medium','low')),
    priority_score INTEGER DEFAULT 200,
    position INTEGER NOT NULL DEFAULT 0,
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    assigned_role TEXT DEFAULT '',
    -- Blocking
    is_blocked INTEGER DEFAULT 0,        -- 1 when in "blocked" column
    blocked_reason TEXT DEFAULT '',
    blocked_at DATETIME DEFAULT NULL,
    blocked_by_agent TEXT DEFAULT '',
    -- Won't Do
    wont_do_requested INTEGER DEFAULT 0, -- 1 when agent requests won't-do (task in "blocked")
    wont_do_reason TEXT DEFAULT '',
    wont_do_requested_by TEXT DEFAULT '',
    wont_do_requested_at DATETIME DEFAULT NULL,
    -- Completion
    completion_summary TEXT DEFAULT '',
    completed_by_agent TEXT DEFAULT '',
    completed_at DATETIME DEFAULT NULL,
    files_modified TEXT DEFAULT '[]',
    -- Work tracking
    resolution TEXT DEFAULT '',          -- NEW: Filled when agent stops work or human moves back
    context_files TEXT DEFAULT '[]',
    tags TEXT DEFAULT '[]',
    estimated_effort TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 9. MCP Tools Changes

**create_task** - Add `summary` param (required):
```
create_task(project_id, title, summary, description?, priority?, ...)
```

**update_task** - Can update `resolution`:
```
update_task(project_id, task_id, resolution="Work done so far...")
```

**block_task** - Moves task to "blocked" column:
```
block_task(project_id, task_id, blocked_reason)
→ sets is_blocked=1, moves to "blocked" column
```

**request_wont_do** - Moves task to "blocked" column:
```
request_wont_do(project_id, task_id, wont_do_reason)
→ sets wont_do_requested=1, moves to "blocked" column
```

### 10. REST API Changes

**POST /api/projects/:id/tasks/:taskId/unblock**
- Resets `is_blocked=0`, clears `blocked_reason`
- Moves task from "blocked" to "todo"
- Adds auto-comment

**POST /api/projects/:id/tasks/:taskId/approve-wont-do**
- Deletes the task (or mark as archived if we add archiving)
- Emits `task_deleted` event

**POST /api/projects/:id/tasks/:taskId/reject-wont-do**
- Resets `wont_do_requested=0`
- Moves task from "blocked" to "todo"
- Requires `reason` in body
- Adds auto-comment with rejection reason

**POST /api/projects/:id/tasks/:taskId/move**
- When moving from "in_progress" to "todo" by human:
  - Append to `resolution` field: "\n\n[Moved back to Todo by human on {date} - task was not completed]"
  - If `resolution` is empty, set it to that message
  - If `resolution` has content, append to end

### 11. WebSocket Events

**Remove:**
- `task_wont_do` (tasks are deleted instead)

**Keep:**
- All other events from PROJECT.md

### 12. UI Changes

**Main Board:**
- 4 columns: Todo | In Progress | Done | Blocked
- Blocked column shows:
  - Tasks with `is_blocked=1` (red indicator, blocked reason)
  - Tasks with `wont_do_requested=1` (orange indicator, "Won't Do Requested" badge)

**Task Modal:**
- Show `summary` field (editable)
- Show `resolution` field (read-only in modal, filled via `update_task` or auto-filled on human move)
- Blocked tasks: "Unblock" button moves to "todo"
- Won't-do-requested tasks: "Approve" (deletes task) / "Reject" (moves to "todo" with comment)

**Task Creation Form:**
- Add required `summary` field (before description)

## Implementation Priority

1. ✅ Domain layer (no Actor, add summary/resolution fields)
2. ✅ SQLite layer (4 columns, new schema)
3. ✅ App layer (blocking logic moves to blocked column, resolution auto-append)
4. ✅ HTTP API (unblock, approve-wont-do moves/deletes correctly)
5. ✅ MCP server (all tools updated)
6. ✅ WebSocket hub
7. ✅ Main server init

## Summary of Key Differences

| Aspect | PROJECT.md | IMPLEMENTATION_SPEC.md |
|--------|------------|------------------------|
| Columns | todo, in_progress, done, wont_do | todo, in_progress, done, **blocked** |
| Blocking | Task stays in todo with flag | Task **moves to blocked column** |
| Won't-do request | Task stays in place with flag | Task **moves to blocked column** |
| Authentication | Not specified | **Explicitly NO auth** |
| Task fields | No summary/resolution | **Added summary (required), resolution** |
| Human moves to todo | Not specified | **Auto-appends to resolution** |
| Priority | 4 levels | **Confirmed: 4 levels only, no boosting** |
| Comment types | Not specified | **Confirmed: no types, just author_type** |
