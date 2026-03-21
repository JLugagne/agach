-- Restrict public schema privileges before creating objects
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    parent_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_by_role TEXT DEFAULT '',
    created_by_agent TEXT DEFAULT '',
    default_role TEXT DEFAULT '',
    owner_user_id TEXT,
    corporation_id TEXT,
    team_id TEXT,
    work_dir TEXT DEFAULT '',
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
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    icon TEXT DEFAULT '',
    color TEXT DEFAULT '#6B7280',
    description TEXT DEFAULT '',
    tech_stack JSONB DEFAULT '[]',
    prompt_hint TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Project-role assignments
CREATE TABLE IF NOT EXISTS project_roles (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    UNIQUE(project_id, role_id)
);

-- Columns table
CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
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
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    column_id TEXT NOT NULL REFERENCES columns(id),
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

-- GIN index on tsvector for fast full-text search
CREATE INDEX IF NOT EXISTS tasks_search_vector_idx ON tasks USING GIN(search_vector);
-- Index for project+column lookups
CREATE INDEX IF NOT EXISTS tasks_project_column_idx ON tasks(project_id, column_id);
-- Index for priority ordering
CREATE INDEX IF NOT EXISTS tasks_project_priority_idx ON tasks(project_id, priority_score DESC, created_at ASC);

-- Comments table
CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY,
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
    id TEXT PRIMARY KEY,
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
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    count INTEGER DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    UNIQUE(project_id, tool_name)
);
