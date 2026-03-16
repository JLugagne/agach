-- Per-project tool usage tracking
-- Tracks MCP tool execution counts per project database

CREATE TABLE IF NOT EXISTS tool_usage (
    tool_name TEXT PRIMARY KEY,
    execution_count INTEGER DEFAULT 0,
    last_executed_at DATETIME DEFAULT NULL
);
