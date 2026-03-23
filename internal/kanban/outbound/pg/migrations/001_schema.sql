-- Restrict public schema privileges before creating objects
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- UUID format validation function
CREATE OR REPLACE FUNCTION is_valid_uuid(val TEXT) RETURNS BOOLEAN AS $$
BEGIN
    RETURN val ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Dockerfiles table: stores versioned Docker Compose service definitions
CREATE TABLE IF NOT EXISTS dockerfiles (
    id          TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version     TEXT NOT NULL,
    content     TEXT NOT NULL DEFAULT '',
    is_latest   BOOLEAN NOT NULL DEFAULT false,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(slug, version)
);

CREATE INDEX IF NOT EXISTS idx_dockerfiles_slug ON dockerfiles (slug);

ALTER TABLE dockerfiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE dockerfiles FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'dockerfiles' AND policyname = 'dockerfiles_all') THEN
        EXECUTE 'CREATE POLICY dockerfiles_all ON dockerfiles USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    parent_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    git_url TEXT DEFAULT '',
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    default_role TEXT DEFAULT '',
    dockerfile_id TEXT REFERENCES dockerfiles(id) ON DELETE SET NULL,
    owner_user_id TEXT,
    corporation_id TEXT,
    team_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'projects' AND policyname = 'projects_all') THEN
        EXECUTE 'CREATE POLICY projects_all ON projects USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Roles table (global)
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    icon TEXT DEFAULT '',
    color TEXT DEFAULT '#6B7280',
    description TEXT DEFAULT '',
    tech_stack JSONB DEFAULT '[]',
    prompt_hint TEXT DEFAULT '',
    prompt_template TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Project-role assignments
CREATE TABLE IF NOT EXISTS project_roles (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    UNIQUE(project_id, role_id)
);

-- Columns table
CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    wip_limit INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, slug)
);

ALTER TABLE columns ENABLE ROW LEVEL SECURITY;
ALTER TABLE columns FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'columns' AND policyname = 'columns_all') THEN
        EXECUTE 'CREATE POLICY columns_all ON columns USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Tasks table with tsvector for full-text search
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    column_id TEXT NOT NULL REFERENCES columns(id),
    feature_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    priority TEXT NOT NULL DEFAULT 'medium' CHECK(priority IN ('critical','high','medium','low')),
    priority_score INTEGER DEFAULT 200,
    position INTEGER NOT NULL DEFAULT 0,
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    assigned_role TEXT DEFAULT '',
    is_blocked INTEGER NOT NULL DEFAULT 0 CHECK(is_blocked IN (0,1)),
    blocked_reason TEXT DEFAULT '',
    blocked_at TIMESTAMPTZ,
    blocked_by_agent TEXT DEFAULT '',
    wont_do_requested INTEGER NOT NULL DEFAULT 0 CHECK(wont_do_requested IN (0,1)),
    wont_do_reason TEXT DEFAULT '',
    wont_do_requested_by TEXT DEFAULT '',
    wont_do_requested_at TIMESTAMPTZ,
    completion_summary TEXT DEFAULT '',
    completed_by_agent TEXT DEFAULT '',
    completed_at TIMESTAMPTZ,
    files_modified JSONB DEFAULT '[]',
    resolution TEXT DEFAULT '',
    context_files JSONB DEFAULT '[]',
    tags JSONB DEFAULT '[]',
    estimated_effort TEXT DEFAULT '',
    seen_by_human BOOLEAN DEFAULT FALSE,
    seen_at TIMESTAMPTZ,
    session_id TEXT NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    model TEXT NOT NULL DEFAULT '',
    cold_start_input_tokens INTEGER NOT NULL DEFAULT 0,
    cold_start_output_tokens INTEGER NOT NULL DEFAULT 0,
    cold_start_cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cold_start_cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    human_estimate_seconds INTEGER NOT NULL DEFAULT 0,
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(summary, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'C')
    ) STORED,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'tasks' AND policyname = 'tasks_all') THEN
        EXECUTE 'CREATE POLICY tasks_all ON tasks USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Task indexes
CREATE INDEX IF NOT EXISTS tasks_search_vector_idx ON tasks USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS tasks_project_column_idx ON tasks(project_id, column_id);
CREATE INDEX IF NOT EXISTS tasks_project_priority_idx ON tasks(project_id, priority_score DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS tasks_feature_id_idx ON tasks(feature_id) WHERE feature_id IS NOT NULL;

-- Comments table
CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_role TEXT NOT NULL DEFAULT '',
    author_name TEXT DEFAULT '',
    author_type TEXT NOT NULL DEFAULT 'agent' CHECK(author_type IN ('agent','human')),
    content TEXT NOT NULL,
    edited_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'comments' AND policyname = 'comments_all') THEN
        EXECUTE 'CREATE POLICY comments_all ON comments USING (true) WITH CHECK (true)';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS comments_task_idx ON comments(task_id);

-- Task dependencies
CREATE TABLE IF NOT EXISTS task_dependencies (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(task_id, depends_on_task_id),
    CHECK(task_id != depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS task_deps_task_idx ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS task_deps_depends_on_idx ON task_dependencies(depends_on_task_id);

-- Tool usage tracking
CREATE TABLE IF NOT EXISTS tool_usage (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    count INTEGER DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    UNIQUE(project_id, tool_name)
);

-- Skills table
CREATE TABLE IF NOT EXISTS skills (
    id          TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
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

ALTER TABLE skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE skills FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'skills' AND policyname = 'skills_all') THEN
        EXECUTE 'CREATE POLICY skills_all ON skills USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Agent-skill join table
CREATE TABLE IF NOT EXISTS agent_skills (
    id         TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    role_id    TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    skill_id   TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, skill_id)
);

ALTER TABLE agent_skills ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_skills FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'agent_skills' AND policyname = 'agent_skills_all') THEN
        EXECUTE 'CREATE POLICY agent_skills_all ON agent_skills USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Model pricing table: stores per-model token pricing rates
CREATE TABLE IF NOT EXISTS model_pricing (
    id                       TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    model_id                 TEXT NOT NULL UNIQUE,
    input_price_per_1m       DOUBLE PRECISION NOT NULL DEFAULT 0,
    output_price_per_1m      DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_read_price_per_1m  DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_write_price_per_1m DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE model_pricing ENABLE ROW LEVEL SECURITY;
ALTER TABLE model_pricing FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'model_pricing' AND policyname = 'model_pricing_all') THEN
        EXECUTE 'CREATE POLICY model_pricing_all ON model_pricing USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Seed default Anthropic model pricing (prices per 1M tokens, as of 2025-05)
INSERT INTO model_pricing (id, model_id, input_price_per_1m, output_price_per_1m, cache_read_price_per_1m, cache_write_price_per_1m)
VALUES
    (gen_random_uuid()::text, 'claude-opus-4-20250514',    15.0, 75.0, 1.5, 18.75),
    (gen_random_uuid()::text, 'claude-sonnet-4-20250514',   3.0, 15.0, 0.3,  3.75),
    (gen_random_uuid()::text, 'claude-haiku-4-20250506',    0.8,  4.0, 0.08, 1.0),
    (gen_random_uuid()::text, 'claude-3-5-sonnet-20241022', 3.0, 15.0, 0.3,  3.75),
    (gen_random_uuid()::text, 'claude-3-5-haiku-20241022',  0.8,  4.0, 0.08, 1.0),
    (gen_random_uuid()::text, 'claude-3-opus-20240229',    15.0, 75.0, 1.5, 18.75),
    (gen_random_uuid()::text, 'claude-3-sonnet-20240229',    3.0, 15.0, 0.3,  3.75),
    (gen_random_uuid()::text, 'claude-3-haiku-20240307',    0.25, 1.25, 0.03, 0.3),
    (gen_random_uuid()::text, 'claude-opus-4-6-20250620',  15.0, 75.0, 1.5, 18.75),
    (gen_random_uuid()::text, 'claude-sonnet-4-6-20250620',  3.0, 15.0, 0.3,  3.75)
ON CONFLICT (model_id) DO NOTHING;

-- Project-agents join table
CREATE TABLE IF NOT EXISTS project_agents (
    id         TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role_id    TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, role_id)
);

ALTER TABLE project_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_agents FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'project_agents' AND policyname = 'project_agents_all') THEN
        EXECUTE 'CREATE POLICY project_agents_all ON project_agents USING (true) WITH CHECK (true)';
    END IF;
END $$;
