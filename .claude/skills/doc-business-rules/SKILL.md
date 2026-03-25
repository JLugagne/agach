---
name: doc-business-rules
description: "Agach business rules: blocking workflow, won't-do workflow, GetNextTask logic, resolution auto-append, priority system, comment system, features"
user-invocable: true
disable-model-invocation: false
---

# Agach Business Rules

## Column Structure (5 Columns)
```
1. backlog       (position -1) - Backlog, created on-demand via EnsureBacklog
2. todo          (position 0)  - Tasks to be done
3. in_progress   (position 1)  - Tasks being worked on
4. done          (position 2)  - Completed tasks
5. blocked       (position 3)  - Tasks requiring human intervention
```

Default columns (todo, in_progress, done, blocked) are created automatically when a project is created.
Backlog column is created on-demand via `EnsureBacklog()`.

## Task Fields

**Required at creation**: `Title` and `Summary`
**Resolution**: Filled when agent stops work or human moves task back to todo

**Key State Flags:**
- `IsBlocked` = true when task is IN the "blocked" column
- `WontDoRequested` = true when task is IN "blocked" column awaiting human decision

**Token Tracking**: `InputTokens`, `OutputTokens`, `CacheReadTokens`, `CacheWriteTokens`, `Model`
**Cold Start Tracking**: `ColdStartInputTokens`, `ColdStartOutputTokens`, etc. (SET semantics, not accumulated)
**Duration Tracking**: `StartedAt`, `DurationSeconds`, `HumanEstimateSeconds`
**Session**: `SessionID` for Claude Code session resuming

## Features
- Features are project-scoped groupings of tasks
- Status: draft → ready → in_progress → done → blocked
- Tasks have optional `FeatureID` linking them to a feature
- Sub-projects were migrated to features (002_features.sql)

## Blocking Workflow
- Agent calls `block_task` → task **moves to "blocked" column**, sets `is_blocked=true`
- Task is visible in Blocked column (not flagged in Todo)
- Only humans can unblock via web UI → moves back to "todo", sets `is_blocked=false`
- Blocked tasks invisible to `get_next_task`

## Won't-Do Workflow
- Agent calls `request_wont_do` → task **moves to "blocked" column** with `wont_do_requested=true`
- Human sees task in Blocked column with "Won't Do Requested" badge
- Human approves → **task moved to "done" column** with `wont_do_requested=true` kept (displays as "won't do")
- Human rejects → adds comment, moves back to "todo", clears `wont_do_requested`
- Won't-do-requested tasks in "blocked" column invisible to `get_next_task`
- Won't-do tasks in "done" column count as resolved dependencies

## GetNextTask Rules
- Returns highest-priority task in "todo" column (not backlog) with all dependencies resolved
- A dependency is "resolved" if its task is in the "done" column (including won't-do approved)
- Filters: `is_blocked=false`, `wont_do_requested=false`
- Role filtering: empty role → unassigned tasks only; specified role → matching or unassigned
- Sorts by `priority_score DESC`, `created_at ASC`
- Optional `featureID` parameter to scope search to a specific feature

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

## Chat Sessions
- Feature-scoped chat sessions that spawn Claude CLI processes on the daemon
- State: active → ended / timeout
- 30-minute idle TTL with 25-minute warning
- Token usage tracked per session (input/output/cache_read/cache_write)
- JSONL capture of full conversation, uploaded to server on session end

## Agent/Role System
- Global agents (roles) with slugs, icons, colors, tech stacks, prompt templates
- Project-scoped agent assignment via `project_agents`
- Specialized agents: variants under a parent agent with specific skill sets
- Skills: global definitions assignable to agents and specialized agents
- Agent cloning: duplicate an existing agent with new slug/name

## Notification Triggers
| Trigger                        | Scope     | Severity  |
|--------------------------------|-----------|-----------|
| Feature status → `done`        | project   | `success` |
| Feature status → `blocked`     | project   | `warning` |
| Task blocked                   | project   | `warning` |
| Task won't-do requested        | project   | `warning` |
| All feature tasks completed    | project   | `success` |
| Agent build failure            | agent     | `error`   |
| System maintenance             | global    | `info`    |
