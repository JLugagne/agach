-- Per-project database schema (<uuid>.db)
-- Each project has its own database file containing its columns, tasks, comments, and dependencies

-- Columns table (4 fixed columns)
CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    wip_limit INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK(slug IN ('todo', 'in_progress', 'done', 'blocked'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_columns_slug ON columns(slug);
CREATE INDEX IF NOT EXISTS idx_columns_position ON columns(position);

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    column_id TEXT NOT NULL REFERENCES columns(id),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    description TEXT DEFAULT '',
    priority TEXT DEFAULT 'medium' CHECK(priority IN ('critical', 'high', 'medium', 'low')),
    priority_score INTEGER DEFAULT 200,
    position INTEGER NOT NULL DEFAULT 0,
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    assigned_role TEXT DEFAULT '',
    is_blocked INTEGER DEFAULT 0,
    blocked_reason TEXT DEFAULT '',
    blocked_at DATETIME DEFAULT NULL,
    blocked_by_agent TEXT DEFAULT '',
    wont_do_requested INTEGER DEFAULT 0,
    wont_do_reason TEXT DEFAULT '',
    wont_do_requested_by TEXT DEFAULT '',
    wont_do_requested_at DATETIME DEFAULT NULL,
    completion_summary TEXT DEFAULT '',
    completed_by_agent TEXT DEFAULT '',
    completed_at DATETIME DEFAULT NULL,
    files_modified TEXT DEFAULT '[]',
    resolution TEXT DEFAULT '',
    context_files TEXT DEFAULT '[]',
    tags TEXT DEFAULT '[]',
    estimated_effort TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasks_column_id ON tasks(column_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assigned_role ON tasks(assigned_role);
CREATE INDEX IF NOT EXISTS idx_tasks_priority_score ON tasks(priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_is_blocked ON tasks(is_blocked);
CREATE INDEX IF NOT EXISTS idx_tasks_wont_do_requested ON tasks(wont_do_requested);

-- Comments table
CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_role TEXT NOT NULL,
    author_name TEXT DEFAULT '',
    author_type TEXT NOT NULL DEFAULT 'agent' CHECK(author_type IN ('agent', 'human')),
    content TEXT NOT NULL,
    edited_at DATETIME DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_task_id ON comments(task_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at);

-- Task dependencies table
CREATE TABLE IF NOT EXISTS task_dependencies (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id, depends_on_task_id),
    CHECK(task_id != depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_dependencies_task_id ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_depends_on_task_id ON task_dependencies(depends_on_task_id);
-- Note: no explicit UNIQUE INDEX needed here; the UNIQUE(task_id, depends_on_task_id) table constraint
-- already causes SQLite to create an implicit index that enforces uniqueness.

-- Insert the 4 fixed columns (OR IGNORE to handle multiple migration runs)
INSERT OR IGNORE INTO columns (id, slug, name, position, wip_limit) VALUES
    ('col_todo', 'todo', 'To Do', 0, 0),
    ('col_in_progress', 'in_progress', 'In Progress', 1, 3),
    ('col_done', 'done', 'Done', 2, 0),
    ('col_blocked', 'blocked', 'Blocked', 3, 0);
