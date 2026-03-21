CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS teams (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash BYTEA NOT NULL DEFAULT '',
    sso_provider  TEXT NOT NULL DEFAULT '',
    sso_subject   BYTEA NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('admin', 'member')),
    team_id       UUID REFERENCES teams(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS users_sso_idx
    ON users(sso_provider, sso_subject)
    WHERE sso_provider <> '';

CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL UNIQUE,
    scopes       JSONB NOT NULL DEFAULT '[]',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS api_keys_user_idx ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS api_keys_hash_idx ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS api_keys_active_idx
    ON api_keys(key_hash)
    WHERE revoked_at IS NULL;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
