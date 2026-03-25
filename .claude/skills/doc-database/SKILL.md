---
name: doc-database
description: "Agach database schema: PostgreSQL tables for projects, roles, tasks, columns, comments, dependencies, features, skills, dockerfiles, notifications, chat sessions, specialized agents, nodes, onboarding"
user-invocable: true
disable-model-invocation: false
---

# Agach Database Schema

PostgreSQL via `github.com/jackc/pgx/v5` + `pgxpool`. Daemon uses SQLite for local build tracking.

## Key Design Choices
- All IDs are UUIDv7 TEXT with `CHECK (is_valid_uuid(id))`
- Row Level Security (RLS) enabled on all tables
- JSONB for arrays (files_modified, context_files, tags, tech_stack)
- TSVECTOR full-text search on tasks (title A, summary B, description C)
- `TIMESTAMPTZ` for all timestamps
- `pgp_sym_encrypt` for sensitive identity columns

## Server Database (`internal/server/outbound/pg/migrations/`)

### 001_schema.sql — Core Tables

#### projects
- `id`, `parent_id` (self-ref CASCADE), `name`, `description`, `git_url`
- `created_by_role`, `created_by_agent`, `default_role`
- `dockerfile_id` (FK dockerfiles), `owner_user_id`, `corporation_id`, `team_id`

#### roles (global agents)
- `id`, `slug` (UNIQUE), `name`, `icon`, `color`, `description`
- `tech_stack` (JSONB), `prompt_hint`, `prompt_template`, `content`, `sort_order`

#### project_roles
- `id`, `project_id` (FK), `role_id` (FK), `sort_order`, UNIQUE(project_id, role_id)

#### columns
- `id`, `project_id` (FK), `slug`, `name`, `position`, `wip_limit`
- UNIQUE(project_id, slug)
- Default columns: backlog(-1), todo(0), in_progress(1), done(2), blocked(3)

#### tasks
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

#### comments
- `id`, `task_id` (FK CASCADE), `author_role`, `author_name`
- `author_type` (agent/human), `content`, `edited_at`

#### task_dependencies
- `id`, `task_id` (FK CASCADE), `depends_on_task_id` (FK CASCADE)
- UNIQUE(task_id, depends_on_task_id), CHECK(task_id != depends_on_task_id)

#### tool_usage
- `id`, `project_id` (FK CASCADE), `tool_name`, `count`, `last_used_at`
- UNIQUE(project_id, tool_name)

#### skills
- `id`, `slug` (UNIQUE), `name`, `description`, `content`, `icon`, `color`, `sort_order`

#### agent_skills
- `id`, `role_id` (FK roles CASCADE), `skill_id` (FK skills CASCADE), `sort_order`
- UNIQUE(role_id, skill_id)

#### model_pricing
- `id`, `model_id` (UNIQUE), input/output/cache_read/cache_write `_price_per_1m`
- Seeded with Anthropic model pricing

#### dockerfiles
- `id`, `slug`, `name`, `description`, `version`, `content`, `is_latest`, `sort_order`
- UNIQUE(slug, version)

#### project_agents
- `id`, `project_id` (FK CASCADE), `role_id` (FK roles CASCADE), `sort_order`
- UNIQUE(project_id, role_id)
- `specialized_agent_id` (FK, added in 005)

### 002_features.sql

#### features
- `id`, `project_id` (FK projects CASCADE), `name`, `description`
- `status` (draft/ready/in_progress/done/blocked)
- `created_by_role`, `created_by_agent`
- Migration converts old sub-projects to features and remaps tasks

### 003_notifications.sql

#### notifications
- `id`, `project_id` (FK projects CASCADE, nullable)
- `scope` (project/agent/global), `agent_slug`
- `severity` (info/success/warning/error)
- `title`, `text`, `link_url`, `link_text`, `link_style`
- `read_at`, `created_at`

### 004_chat_sessions.sql

#### chat_sessions
- `id`, `feature_id` (FK features CASCADE), `project_id` (FK projects CASCADE)
- `state` (active/ended/timeout), `claude_session_id`, `jsonl_path`
- Token counts: `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`, `model`
- `started_at`, `ended_at`

### 005_specialized_agents.sql

#### specialized_agents
- `id`, `parent_agent_id` (FK roles CASCADE), `slug` (UNIQUE), `name`, `sort_order`

#### specialized_agent_skills
- `id`, `specialized_agent_id` (FK CASCADE), `skill_id` (FK skills CASCADE)
- UNIQUE(specialized_agent_id, skill_id)

### 006_chat_sessions_node_id.sql
- Adds `node_id` column to chat_sessions

## Identity Database (`internal/identity/outbound/pg/migrations/`)

### 001_identity.sql

#### teams
- `id`, `name`, `slug` (UNIQUE), `description`, `created_at`, `updated_at`

#### users
- `id`, `email` (UNIQUE), `display_name`, `password_hash` (encrypted via pgp_sym_encrypt)
- `sso_provider`, `sso_subject` (bytea, encrypted)
- `role` (admin/member), `team_id` (FK teams)
- Auto-update trigger on `updated_at`

### 002_nodes_onboarding.sql

#### nodes
- `id`, `owner_user_id` (FK users CASCADE), `name`
- `mode` (default/shared), `status` (active/revoked)
- `refresh_token_hash`, `last_seen_at`, `revoked_at`

#### onboarding_codes
- `id`, `code` (6-digit numeric), `created_by_user_id` (FK users CASCADE)
- `node_mode`, `node_name`, `expires_at`, `used_at`, `used_by_node_id`

#### node_access
- `id`, `node_id` (FK nodes CASCADE)
- `user_id` (FK users CASCADE, nullable), `team_id` (FK teams CASCADE, nullable)
- CHECK: at least one of user_id or team_id must be set

## Daemon SQLite (`internal/daemon/outbound/sqlite/migrations/`)

### 001_builds.sql

#### builds
- `id` (TEXT PK), `dockerfile_slug`, `version`
- `image_hash`, `image_size` (INTEGER), `status` (pending/building/success/failed)
- `build_log`, `created_at`, `completed_at`
- UNIQUE(dockerfile_slug, version)
- Indexes on dockerfile_slug and status
