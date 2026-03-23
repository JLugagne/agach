-- Create features table
CREATE TABLE IF NOT EXISTS features (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','ready','in_progress','done','blocked')),
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_features_project_id ON features(project_id);
CREATE INDEX IF NOT EXISTS idx_features_project_status ON features(project_id, status);

ALTER TABLE features ENABLE ROW LEVEL SECURITY;
ALTER TABLE features FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'features' AND policyname = 'features_all') THEN
        EXECUTE 'CREATE POLICY features_all ON features USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Migrate existing sub-project features to features table
-- 1. Insert features from sub-projects (keep same UUIDs)
INSERT INTO features (id, project_id, name, description, status, created_by_role, created_by_agent, created_at, updated_at)
SELECT p.id, p.parent_id, p.name, COALESCE(p.description, ''), 'draft', COALESCE(p.created_by_role, ''), COALESCE(p.created_by_agent, ''), p.created_at, p.updated_at
FROM projects p
WHERE p.parent_id IS NOT NULL
ON CONFLICT (id) DO NOTHING;

-- 2. For tasks in sub-project boards, remap them to the parent project
-- Move tasks: set project_id to the parent project, set feature_id to the sub-project id (now a feature id)
-- Also remap column_id to matching column in parent project (by slug)
UPDATE tasks t
SET
    project_id = p.parent_id,
    feature_id = t.project_id,
    column_id = (
        SELECT pc.id FROM columns pc
        WHERE pc.project_id = p.parent_id
        AND pc.slug = (SELECT c.slug FROM columns c WHERE c.id = t.column_id)
        LIMIT 1
    )
FROM projects p
WHERE t.project_id = p.id
AND p.parent_id IS NOT NULL;

-- 3. Drop old FK on tasks.feature_id (references projects), recreate referencing features
-- First drop any existing FK
DO $$ BEGIN
    ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_feature_id_fkey;
EXCEPTION WHEN undefined_object THEN
    NULL;
END $$;

-- Drop the partial index on feature_id if it exists
DROP INDEX IF EXISTS tasks_feature_id_idx;

-- Add new FK referencing features table
ALTER TABLE tasks ADD CONSTRAINT tasks_feature_id_fkey
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE SET NULL;

-- Recreate partial index
CREATE INDEX IF NOT EXISTS tasks_feature_id_idx ON tasks(feature_id) WHERE feature_id IS NOT NULL;

-- 4. Delete old sub-project rows (their columns will cascade delete, tasks already moved)
-- First delete their columns
DELETE FROM columns WHERE project_id IN (SELECT id FROM projects WHERE parent_id IS NOT NULL);
-- Then delete the sub-projects themselves
DELETE FROM projects WHERE parent_id IS NOT NULL;
