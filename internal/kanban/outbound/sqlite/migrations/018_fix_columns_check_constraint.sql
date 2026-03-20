-- Rebuild the columns table with a CHECK constraint that includes 'backlog'.
-- Project DBs created before migration 016 have CHECK(slug IN ('todo', 'in_progress', 'done', 'blocked')).
-- INSERT OR IGNORE in migration 016 silently skips 'backlog' when the constraint rejects it.
-- This migration rebuilds the table idempotently without depending on the old table existing.

PRAGMA foreign_keys = OFF;

-- Clean up any leftover columns_new from a failed previous run.
DROP TABLE IF EXISTS columns_new;

CREATE TABLE columns_new (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    wip_limit INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK(slug IN ('backlog', 'todo', 'in_progress', 'done', 'blocked'))
);

-- Seed all default columns (no dependency on the old table existing).
-- Custom WIP limits cannot be preserved without SELECT FROM columns, and the old table
-- may not exist in a partial-migration recovery scenario.
INSERT OR IGNORE INTO columns_new (id, slug, name, position, wip_limit) VALUES
    ('col_backlog', 'backlog', 'Backlog', -1, 0),
    ('col_todo', 'todo', 'To Do', 0, 0),
    ('col_in_progress', 'in_progress', 'In Progress', 1, 3),
    ('col_done', 'done', 'Done', 2, 0),
    ('col_blocked', 'blocked', 'Blocked', 3, 0);

-- Drop old table (IF EXISTS: safe if it was already dropped in a partial run).
DROP TABLE IF EXISTS columns;

ALTER TABLE columns_new RENAME TO columns;

CREATE UNIQUE INDEX IF NOT EXISTS idx_columns_slug ON columns(slug);
CREATE INDEX IF NOT EXISTS idx_columns_position ON columns(position);

PRAGMA foreign_keys = ON;
