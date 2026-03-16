-- Full-text search index for tasks using FTS5
-- Indexes title, summary, and description for efficient text search

CREATE VIRTUAL TABLE IF NOT EXISTS tasks_fts USING fts5(
    title,
    summary,
    description,
    tags,
    content='tasks',
    content_rowid='rowid'
);

-- Rebuild FTS index from existing tasks (idempotent — safe to re-run)
INSERT INTO tasks_fts(tasks_fts) VALUES('rebuild');

-- Keep FTS in sync: INSERT trigger
CREATE TRIGGER IF NOT EXISTS tasks_fts_insert AFTER INSERT ON tasks BEGIN
    INSERT INTO tasks_fts(rowid, title, summary, description, tags)
    VALUES (new.rowid, new.title, new.summary, new.description, new.tags);
END;

-- Keep FTS in sync: DELETE trigger
CREATE TRIGGER IF NOT EXISTS tasks_fts_delete AFTER DELETE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, summary, description, tags)
    VALUES ('delete', old.rowid, old.title, old.summary, old.description, old.tags);
END;

-- Keep FTS in sync: UPDATE trigger
CREATE TRIGGER IF NOT EXISTS tasks_fts_update AFTER UPDATE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, summary, description, tags)
    VALUES ('delete', old.rowid, old.title, old.summary, old.description, old.tags);
    INSERT INTO tasks_fts(rowid, title, summary, description, tags)
    VALUES (new.rowid, new.title, new.summary, new.description, new.tags);
END;
