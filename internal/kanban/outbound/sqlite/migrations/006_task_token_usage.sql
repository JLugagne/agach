-- Add token usage tracking fields to tasks
ALTER TABLE tasks ADD COLUMN input_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN output_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN cache_read_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN cache_write_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN model TEXT DEFAULT '';
