-- Migration: nodes and onboarding_codes tables

CREATE TABLE IF NOT EXISTS nodes (
    id                 UUID PRIMARY KEY,
    owner_user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    mode               TEXT NOT NULL CHECK(mode IN ('default', 'shared')),
    status             TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'revoked')),
    refresh_token_hash BYTEA NOT NULL,
    last_seen_at       TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nodes_owner ON nodes(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status) WHERE status = 'active';

CREATE OR REPLACE TRIGGER nodes_updated_at
    BEFORE UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS onboarding_codes (
    id                 UUID PRIMARY KEY,
    code               TEXT NOT NULL UNIQUE,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    node_mode          TEXT NOT NULL CHECK(node_mode IN ('default', 'shared')),
    node_name          TEXT NOT NULL DEFAULT '',
    expires_at         TIMESTAMPTZ NOT NULL,
    used_at            TIMESTAMPTZ,
    used_by_node_id    UUID REFERENCES nodes(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_onboarding_codes_code ON onboarding_codes(code) WHERE used_at IS NULL;

CREATE TABLE IF NOT EXISTS node_access (
    id         UUID PRIMARY KEY,
    node_id    UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    team_id    UUID REFERENCES teams(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (user_id IS NOT NULL AND team_id IS NULL) OR
        (user_id IS NULL AND team_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_node_access_user ON node_access(node_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_node_access_team ON node_access(node_id, team_id) WHERE team_id IS NOT NULL;
