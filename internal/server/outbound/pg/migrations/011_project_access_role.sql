-- Add role column to project_user_access if missing (may have been created by
-- an earlier version of 010 that lacked the column).

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'project_user_access' AND column_name = 'role'
    ) THEN
        ALTER TABLE project_user_access
            ADD COLUMN role TEXT NOT NULL DEFAULT 'member';
        ALTER TABLE project_user_access
            ADD CONSTRAINT project_user_access_role_check CHECK (role IN ('admin', 'member'));
    END IF;
END
$$;
