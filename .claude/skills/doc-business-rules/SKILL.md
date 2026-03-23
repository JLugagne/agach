---
name: doc-business-rules
description: "Agach business rules: blocking workflow, won't-do workflow, GetNextTask logic, resolution auto-append, priority system, comment system"
user-invocable: true
disable-model-invocation: false
---

# Agach Business Rules

## Column Structure (4 Fixed Columns)
```
1. todo          (position 0) - Tasks to be done
2. in_progress   (position 1) - Tasks being worked on
3. done          (position 2) - Completed tasks
4. blocked       (position 3) - Tasks requiring human intervention
```

Changed from PROJECT.md: `wont_do` column replaced with `blocked` column.

## Task Fields

**Required Fields:**
- `Summary` (string) - Brief description, required at creation
- `Resolution` (string) - Filled when agent stops work or human moves task back to todo

**Key State Flags:**
- `IsBlocked` = 1 when task is IN the "blocked" column (not just flagged)
- `WontDoRequested` = 1 when task is IN "blocked" column awaiting human decision
- `WontDoConfirmed` removed (tasks are deleted instead of confirmed)

## Blocking Workflow
- Agent calls `block_task` → task **moves to "blocked" column**, sets `is_blocked=1`
- Task is visible in Blocked column (not flagged in Todo)
- Only humans can unblock via web UI → moves back to "todo", sets `is_blocked=0`
- Blocked tasks invisible to `get_next_task`

## Won't-Do Workflow
- Agent calls `request_wont_do` → task **moves to "blocked" column** with `wont_do_requested=1`
- Human sees task in Blocked column with "Won't Do Requested" badge
- Human approves → **task moved to "done" column** with `wont_do_requested=1` kept as state marker (displays as "won't do")
- Human rejects → adds comment, moves back to "todo", clears `wont_do_requested`
- Won't-do-requested tasks in "blocked" column invisible to `get_next_task`
- Won't-do tasks in "done" column count as resolved dependencies

## GetNextTask Rules
- Returns highest-priority task in "todo" column with all dependencies resolved
- A dependency is "resolved" if its task is in the "done" column (including won't-do approved tasks)
- Filters: `is_blocked=0`, `wont_do_requested=0`, `assigned_role` matches or empty
- Sorts by `priority_score DESC`, `created_at ASC`
- Optional `sub_project_id` parameter scopes search to that sub-project and all its descendants

## Resolution Auto-Append
When human moves task from "in_progress" to "todo":
```go
if resolution == "" {
    resolution = "[Moved back to Todo by human on {date} - task was not completed]"
} else {
    resolution += "\n\n[Moved back to Todo by human on {date} - task was not completed]"
}
```

## Priority System
Simple 4-level system only:
- `critical` → 400
- `high` → 300
- `medium` → 200
- `low` → 100

**NO dynamic boosting, NO urgency calculations, NO dependency boosting**

## Comment System
- `author_type` field: "agent" or "human"
- No comment types/categories
- Auto-comments for system actions (blocking, won't-do, etc.)
