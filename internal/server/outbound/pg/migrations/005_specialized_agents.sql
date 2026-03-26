CREATE TABLE IF NOT EXISTS specialized_agents (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    parent_agent_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE specialized_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE specialized_agents FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'specialized_agents' AND policyname = 'specialized_agents_all') THEN
        EXECUTE 'CREATE POLICY specialized_agents_all ON specialized_agents USING (true) WITH CHECK (true)';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS specialized_agent_skills (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    specialized_agent_id TEXT NOT NULL REFERENCES specialized_agents(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(specialized_agent_id, skill_id)
);

ALTER TABLE specialized_agent_skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE specialized_agent_skills FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'specialized_agent_skills' AND policyname = 'specialized_agent_skills_all') THEN
        EXECUTE 'CREATE POLICY specialized_agent_skills_all ON specialized_agent_skills USING (true) WITH CHECK (true)';
    END IF;
END $$;

ALTER TABLE project_agents ADD COLUMN IF NOT EXISTS specialized_agent_id TEXT REFERENCES specialized_agents(id);
