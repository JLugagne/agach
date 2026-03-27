-- Project access control: grant users and teams access to specific projects.

CREATE TABLE IF NOT EXISTS project_user_access (
    id          TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, user_id)
);

CREATE TABLE IF NOT EXISTS project_team_access (
    id          TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    team_id     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_project_user_access_user ON project_user_access(user_id);
CREATE INDEX IF NOT EXISTS idx_project_team_access_team ON project_team_access(team_id);

ALTER TABLE project_user_access ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_team_access ENABLE ROW LEVEL SECURITY;
