CREATE TABLE IF NOT EXISTS builds (
    id              TEXT PRIMARY KEY,
    dockerfile_slug TEXT NOT NULL,
    version         TEXT NOT NULL,
    image_hash      TEXT,
    image_size      INTEGER,
    status          TEXT NOT NULL DEFAULT 'pending',
    build_log       TEXT,
    created_at      TEXT NOT NULL,
    completed_at    TEXT,
    UNIQUE(dockerfile_slug, version)
);

CREATE INDEX IF NOT EXISTS idx_builds_dockerfile_slug ON builds(dockerfile_slug);
CREATE INDEX IF NOT EXISTS idx_builds_status ON builds(status);
