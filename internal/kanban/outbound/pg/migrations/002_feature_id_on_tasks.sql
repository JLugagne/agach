-- Migration 002: add feature_id column to tasks
-- A feature is a project with parent_id IS NOT NULL.
-- feature_id links a task directly to its feature scope for efficient filtering.
-- Nullable: tasks at root-project level have feature_id = NULL.
-- ON DELETE CASCADE: deleting a feature project cascades to its tasks via project_id already;
--   feature_id references the same projects table and also cascades for FK integrity.

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS feature_id TEXT REFERENCES projects(id) ON DELETE SET NULL;

-- Index for efficient "list tasks by feature" queries
CREATE INDEX IF NOT EXISTS tasks_feature_id_idx ON tasks(feature_id) WHERE feature_id IS NOT NULL;
