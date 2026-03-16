-- Per-project database index optimizations for existing databases

-- === TASKS ===

-- Composite covering index for GetNextTask (hottest query path)
-- Covers: WHERE column_id=? AND is_blocked=0 AND wont_do_requested=0 ORDER BY priority_score DESC, created_at ASC
CREATE INDEX IF NOT EXISTS idx_tasks_get_next ON tasks(column_id, is_blocked, wont_do_requested, priority_score DESC, created_at ASC);

-- Composite index for List with column filter + sort
-- Covers: WHERE column_id=? ORDER BY priority_score DESC, created_at ASC
CREATE INDEX IF NOT EXISTS idx_tasks_column_priority ON tasks(column_id, priority_score DESC, created_at ASC);

-- Index for UpdatedSince filter (incremental sync polling)
CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at);

-- Drop single-column indexes now made redundant by composites above
DROP INDEX IF EXISTS idx_tasks_column_id;
DROP INDEX IF EXISTS idx_tasks_is_blocked;
DROP INDEX IF EXISTS idx_tasks_wont_do_requested;
DROP INDEX IF EXISTS idx_tasks_priority_score;

-- === COMMENTS ===

-- Composite index for List + IsLastComment: covers WHERE task_id=? ORDER BY created_at
-- Also serves Count (WHERE task_id=?) as a prefix match
-- Replaces idx_comments_task_id and idx_comments_created_at
CREATE INDEX IF NOT EXISTS idx_comments_task_id_created_at ON comments(task_id, created_at ASC);

DROP INDEX IF EXISTS idx_comments_task_id;
DROP INDEX IF EXISTS idx_comments_created_at;

-- === TASK DEPENDENCIES ===

-- Composite index for List + GetDependencyContext: covers WHERE task_id=? ORDER BY created_at
-- Replaces idx_task_dependencies_task_id
CREATE INDEX IF NOT EXISTS idx_task_deps_task_id_created_at ON task_dependencies(task_id, created_at ASC);

-- Composite index for ListDependents: covers WHERE depends_on_task_id=? ORDER BY created_at
-- Replaces idx_task_dependencies_depends_on_task_id
CREATE INDEX IF NOT EXISTS idx_task_deps_depends_on_created_at ON task_dependencies(depends_on_task_id, created_at ASC);

DROP INDEX IF EXISTS idx_task_dependencies_task_id;
DROP INDEX IF EXISTS idx_task_dependencies_depends_on_task_id;
