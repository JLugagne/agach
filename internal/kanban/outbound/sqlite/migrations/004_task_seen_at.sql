-- Add seen_at to track when a task was first viewed
-- NULL = unseen, non-NULL = timestamp of first view
ALTER TABLE tasks ADD COLUMN seen_at DATETIME DEFAULT NULL;
