-- Add node_id to tasks and features so we can list them per node.
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS node_id TEXT;
ALTER TABLE features ADD COLUMN IF NOT EXISTS node_id TEXT;

CREATE INDEX IF NOT EXISTS idx_tasks_node_id ON tasks (node_id) WHERE node_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_features_node_id ON features (node_id) WHERE node_id IS NOT NULL;
