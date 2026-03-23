---
name: doc-database
description: "Agach database schema: SQLite tables for projects, roles, tasks, columns, comments, dependencies — global DB and per-project DB"
user-invocable: true
disable-model-invocation: false
---

# Agach Database Schema

Uses SQLite (driver: `github.com/mattn/go-sqlite3`). The outbound adapter is named `pg/` historically.

## Global DB (`kanban.db`)
```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    parent_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    icon TEXT DEFAULT '',
    color TEXT DEFAULT '#6B7280',
    description TEXT DEFAULT '',
    tech_stack TEXT DEFAULT '[]',
    prompt_hint TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Per-Project DB (`<uuid>.db`)
```sql
CREATE TABLE columns (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,  -- "todo", "in_progress", "done", "blocked"
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    column_id TEXT NOT NULL REFERENCES columns(id),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,                -- Required at creation
    description TEXT DEFAULT '',
    priority TEXT DEFAULT 'medium' CHECK(priority IN ('critical','high','medium','low')),
    priority_score INTEGER DEFAULT 200,
    position INTEGER NOT NULL DEFAULT 0,
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    assigned_role TEXT DEFAULT '',
    is_blocked INTEGER DEFAULT 0,         -- 1 when in "blocked" column
    blocked_reason TEXT DEFAULT '',
    blocked_at DATETIME DEFAULT NULL,
    blocked_by_agent TEXT DEFAULT '',
    wont_do_requested INTEGER DEFAULT 0,  -- 1 when agent requests won't-do
    wont_do_reason TEXT DEFAULT '',
    wont_do_requested_by TEXT DEFAULT '',
    wont_do_requested_at DATETIME DEFAULT NULL,
    completion_summary TEXT DEFAULT '',
    completed_by_agent TEXT DEFAULT '',
    completed_at DATETIME DEFAULT NULL,
    files_modified TEXT DEFAULT '[]',
    resolution TEXT DEFAULT '',           -- Filled when agent stops work
    context_files TEXT DEFAULT '[]',
    tags TEXT DEFAULT '[]',
    estimated_effort TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE comments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_role TEXT NOT NULL,
    author_name TEXT DEFAULT '',
    author_type TEXT NOT NULL DEFAULT 'agent' CHECK(author_type IN ('agent', 'human')),
    content TEXT NOT NULL,
    edited_at DATETIME DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_dependencies (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id, depends_on_task_id),
    CHECK(task_id != depends_on_task_id)
);
```
