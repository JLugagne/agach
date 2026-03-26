ALTER TABLE chat_sessions ADD COLUMN IF NOT EXISTS node_id TEXT;
CREATE INDEX IF NOT EXISTS idx_chat_sessions_node_id ON chat_sessions(node_id);
