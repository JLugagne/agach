-- Migration: chat_sessions table
-- Tracks chat sessions between users and Claude for features

CREATE TABLE IF NOT EXISTS chat_sessions (
    id TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'ended', 'timeout')),
    claude_session_id TEXT,
    jsonl_path TEXT,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    model TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_feature_id ON chat_sessions(feature_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_project_id ON chat_sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_state ON chat_sessions(state);

-- Enable RLS
ALTER TABLE chat_sessions ENABLE ROW LEVEL SECURITY;
