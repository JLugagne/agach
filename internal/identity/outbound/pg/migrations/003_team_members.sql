-- Many-to-many team membership (replaces users.team_id one-to-one)

CREATE TABLE IF NOT EXISTS team_members (
    team_id    UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_id, user_id)
);

-- Migrate existing one-to-one data (only if column still exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'team_id'
    ) THEN
        INSERT INTO team_members (team_id, user_id, created_at)
        SELECT team_id, id, NOW()
        FROM users
        WHERE team_id IS NOT NULL
        ON CONFLICT DO NOTHING;

        ALTER TABLE users DROP COLUMN team_id;
    END IF;
END $$;
