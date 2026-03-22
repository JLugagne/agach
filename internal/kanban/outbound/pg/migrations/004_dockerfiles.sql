-- Dockerfiles table: stores versioned Docker Compose service definitions
CREATE TABLE IF NOT EXISTS dockerfiles (
    id          TEXT PRIMARY KEY,
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

-- Index for slug lookups
CREATE INDEX IF NOT EXISTS idx_dockerfiles_slug ON dockerfiles (slug);

-- RLS
ALTER TABLE dockerfiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE dockerfiles FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'dockerfiles' AND policyname = 'dockerfiles_all'
    ) THEN
        EXECUTE 'CREATE POLICY dockerfiles_all ON dockerfiles USING (true) WITH CHECK (true)';
    END IF;
END $$;

-- Add dockerfile_id FK to projects (one project → one dockerfile, nullable)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'projects' AND column_name = 'dockerfile_id'
    ) THEN
        ALTER TABLE projects ADD COLUMN dockerfile_id TEXT REFERENCES dockerfiles(id) ON DELETE SET NULL;
    END IF;
END $$;
