---
name: doc-database
description: "Agach database schema: PostgreSQL tables for projects, roles, tasks, columns, comments, dependencies, features, skills, dockerfiles, model pricing"
user-invocable: true
disable-model-invocation: false
---

# Agach Database Schema

PostgreSQL via `github.com/jackc/pgx/v5` + `pgxpool`. Outbound adapter is at `internal/kanban/outbound/pg/`.
Migrations in `internal/kanban/outbound/pg/migrations/`.

## Key Design Choices
- All IDs are UUIDv7 TEXT with `CHECK (is_valid_uuid(id))`
- Row Level Security (RLS) enabled on all tables
- JSONB for arrays (files_modified, context_files, tags, tech_stack)
- TSVECTOR full-text search on tasks (title A, summary B, description C)
- `TIMESTAMPTZ` for all timestamps

## Tables (001_schema.sql)

### projects
- `id`, `parent_id` (self-ref CASCADE), `name`, `description`, `git_url`
- `created_by_role`, `created_by_agent`, `default_role`
- `dockerfile_id` (FK dockerfiles), `owner_user_id`, `corporation_id`, `team_id`

### roles (global)
- `id`, `slug` (UNIQUE), `name`, `icon`, `color`, `description`
- `tech_stack` (JSONB), `prompt_hint`, `prompt_template`, `content`, `sort_order`

### project_roles
- `id`, `project_id` (FK), `role_id` (FK), `sort_order`, UNIQUE(project_id, role_id)

### columns
- `id`, `project_id` (FK), `slug`, `name`, `position`, `wip_limit`
- UNIQUE(project_id, slug)
- Default columns: backlog(-1), todo(0), in_progress(1), done(2), blocked(3)

### tasks
- Core: `id`, `project_id` (FK), `column_id` (FK), `feature_id` (FK features), `title`, `summary`, `description`
- Priority: `priority` (critical/high/medium/low), `priority_score`
- Assignment: `position`, `created_by_role`, `created_by_agent`, `assigned_role`
- Blocking: `is_blocked` (0/1), `blocked_reason`, `blocked_at`, `blocked_by_agent`
- Won't-do: `wont_do_requested` (0/1), `wont_do_reason`, `wont_do_requested_by`, `wont_do_requested_at`
- Completion: `completion_summary`, `completed_by_agent`, `completed_at`
- Files: `files_modified` (JSONB), `context_files` (JSONB)
- Resolution: `resolution`, `tags` (JSONB), `estimated_effort`
- Tracking: `seen_by_human`, `seen_at`, `session_id`
- Tokens: `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`, `model`
- Cold start: `cold_start_input_tokens`, `cold_start_output_tokens`, `cold_start_cache_read_tokens`, `cold_start_cache_write_tokens`
- Duration: `started_at`, `duration_seconds`, `human_estimate_seconds`
- Search: `search_vector` (TSVECTOR GENERATED STORED)

### comments
- `id`, `task_id` (FK CASCADE), `author_role`, `author_name`
- `author_type` (agent/human), `content`, `edited_at`

### task_dependencies
- `id`, `task_id` (FK CASCADE), `depends_on_task_id` (FK CASCADE)
- UNIQUE(task_id, depends_on_task_id), CHECK(task_id != depends_on_task_id)

### tool_usage
- `id`, `project_id` (FK CASCADE), `tool_name`, `count`, `last_used_at`
- UNIQUE(project_id, tool_name)

### skills
- `id`, `slug` (UNIQUE), `name`, `description`, `content`, `icon`, `color`, `sort_order`

### agent_skills
- `id`, `role_id` (FK roles CASCADE), `skill_id` (FK skills CASCADE), `sort_order`
- UNIQUE(role_id, skill_id)

### model_pricing
- `id`, `model_id` (UNIQUE), input/output/cache_read/cache_write `_price_per_1m`
- Seeded with Anthropic model pricing

### dockerfiles
- `id`, `slug`, `name`, `description`, `version`, `content`, `is_latest`, `sort_order`
- UNIQUE(slug, version)

### project_agents
- `id`, `project_id` (FK CASCADE), `role_id` (FK roles CASCADE), `sort_order`
- UNIQUE(project_id, role_id)

## Tables (002_features.sql)

### features
- `id`, `project_id` (FK projects CASCADE), `name`, `description`
- `status` (draft/ready/in_progress/done/blocked)
- `created_by_role`, `created_by_agent`
- Migration converts old sub-projects to features and remaps tasks
