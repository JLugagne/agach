-- 003_notifications.sql: Notification system
CREATE TABLE IF NOT EXISTS notifications (
    id          TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    severity    TEXT NOT NULL CHECK (severity IN ('info', 'success', 'warning', 'error')),
    title       TEXT NOT NULL,
    text        TEXT NOT NULL,
    link_url    TEXT NOT NULL DEFAULT '',
    link_text   TEXT NOT NULL DEFAULT '',
    link_style  TEXT NOT NULL DEFAULT '',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_project_unread ON notifications (project_id, created_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_project_created ON notifications (project_id, created_at DESC);
