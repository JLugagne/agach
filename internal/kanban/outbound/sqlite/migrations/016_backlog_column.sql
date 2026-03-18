-- Add backlog column (position -1, before todo)
INSERT OR IGNORE INTO columns (id, slug, name, position, wip_limit) VALUES
    ('col_backlog', 'backlog', 'Backlog', -1, 0);
