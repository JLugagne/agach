-- Migration: feature changelogs

ALTER TABLE features ADD COLUMN IF NOT EXISTS user_changelog TEXT DEFAULT '';
ALTER TABLE features ADD COLUMN IF NOT EXISTS tech_changelog TEXT DEFAULT '';
