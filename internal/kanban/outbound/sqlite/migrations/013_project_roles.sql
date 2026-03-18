CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    icon TEXT DEFAULT '',
    color TEXT DEFAULT '#6B7280',
    description TEXT DEFAULT '',
    tech_stack TEXT DEFAULT '[]',
    prompt_hint TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_proj_roles_slug ON roles(slug);
CREATE INDEX IF NOT EXISTS idx_proj_roles_sort_order ON roles(sort_order);
