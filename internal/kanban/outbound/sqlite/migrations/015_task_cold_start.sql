ALTER TABLE tasks ADD COLUMN cold_start_input_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN cold_start_output_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN cold_start_cache_read_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN cold_start_cache_write_tokens INTEGER DEFAULT 0;
