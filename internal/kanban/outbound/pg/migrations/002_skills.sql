-- Skills table
CREATE TABLE IF NOT EXISTS skills (
    id          TEXT PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL DEFAULT '',
    icon        TEXT NOT NULL DEFAULT '',
    color       TEXT NOT NULL DEFAULT '#6B7280',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RLS
ALTER TABLE skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE skills FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'skills' AND policyname = 'skills_all'
    ) THEN
        EXECUTE 'CREATE POLICY skills_all ON skills USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Agent-skill join table
CREATE TABLE IF NOT EXISTS agent_skills (
    id         TEXT PRIMARY KEY,
    role_id    TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    skill_id   TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, skill_id)
);

ALTER TABLE agent_skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_skills FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'agent_skills' AND policyname = 'agent_skills_all'
    ) THEN
        EXECUTE 'CREATE POLICY agent_skills_all ON agent_skills USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Add content column to roles if not already present
ALTER TABLE roles ADD COLUMN IF NOT EXISTS content TEXT NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS prompt_template TEXT NOT NULL DEFAULT '';

-- Project-agents join table (replaces copy model)
CREATE TABLE IF NOT EXISTS project_agents (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role_id    TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, role_id)
);

ALTER TABLE project_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_agents FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'project_agents' AND policyname = 'project_agents_all'
    ) THEN
        EXECUTE 'CREATE POLICY project_agents_all ON project_agents USING (true) WITH CHECK (true)';
    END IF;
END $$;
