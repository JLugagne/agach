-- Global database index optimizations for existing databases
-- Composite index for ListByWorkDir: covers WHERE work_dir = ? ORDER BY created_at DESC
CREATE INDEX IF NOT EXISTS idx_projects_work_dir_created_at ON projects(work_dir, created_at DESC);

-- Composite index for List: covers WHERE parent_id = ? ORDER BY created_at DESC
-- Also serves CountChildren (WHERE parent_id = ?) as a prefix match
-- Replaces idx_projects_parent_id
DROP INDEX IF EXISTS idx_projects_parent_id;
CREATE INDEX IF NOT EXISTS idx_projects_parent_id_created_at ON projects(parent_id, created_at DESC);
