---
name: doc-architecture
description: "Agach project hexagonal architecture: bounded contexts, source code hierarchy, entry points, layers, and package structure"
user-invocable: true
disable-model-invocation: false
---

# Agach Architecture — Hexagonal / Clean Architecture

## Entry Points (`cmd/`)
```
cmd/agach/main.go           - CLI client (TUI monitor)
cmd/agach-server/main.go    - HTTP + MCP server entry point (PostgreSQL via pgxpool)
cmd/qa-seed/main.go         - Seeds database with test data for QA
```

## Backend — Hexagonal Architecture (`internal/`)

Each bounded context follows: `domain/` → `app/` → `inbound/` + `outbound/`

### `internal/server/` — Core server bounded context
```
domain/
  types.go                  - Task, Column, Comment, Project, Role, Feature domain types
  errors.go                 - Domain errors (ErrNotFound, ErrInvalidMove, etc.)
  service/
    commands.go             - Command service interface (write operations)
    queries.go              - Query service interface (read operations)
    servicetest/contract.go - Service mock + contract tests
  repositories/             - Repository interfaces + test contracts
    tasks/                  - Task repository (CRUD, GetNextTask, move, block)
    columns/                - Column repository (5 columns: backlog, todo, in_progress, done, blocked)
    projects/               - Project repository
    comments/               - Comment repository
    dependencies/           - Task dependency repository
    features/               - Feature repository
    agents/                 - Agent management repository
    skills/                 - Skill repository
    dockerfiles/            - Dockerfile repository
    toolusage/              - Tool usage tracking repository
    unitofwork.go           - Unit of Work interface
    Each has: <name>.go (interface) + <name>test/contract.go (mock + contract)

app/
  app.go                    - Application service (wires domain logic)
  tasks.go                  - Task operations (create, move, block, complete, get_next)
  columns.go                - Column operations
  comments.go               - Comment operations
  dependencies.go           - Dependency operations
  roles.go                  - Role CRUD
  skills.go                 - Skill CRUD
  dockerfiles.go            - Dockerfile management
  prompt.go                 - Prompt template generation
  agent_management.go       - Agent lifecycle management
  task_move_project.go      - Cross-project task moves
  tool_usage.go             - Tool usage tracking
  *_test.go                 - Unit tests using mock repositories

inbound/
  commands/                 - HTTP command handlers (POST/PUT/DELETE via gorilla/mux)
    router.go               - Route registration
    tasks.go                - Task command handlers
    comments.go             - Comment handlers
    projects.go             - Project handlers
    roles.go                - Role handlers
    skills.go               - Skill handlers
    features.go             - Feature handlers
    dockerfiles.go          - Dockerfile handlers
    project_agents.go       - Agent management handlers
    seen.go                 - Task seen tracking
    images.go               - Image upload
  converters/               - Request/response type converters (API ↔ domain)
    tasks.go, columns.go, comments.go, projects.go, roles.go, features.go, etc.
  queries/                  - HTTP query handlers (GET via gorilla/mux)
    router.go               - Query route registration
    tasks.go                - Task queries (list, get, board, next_task)
    projects.go             - Project queries
    comments.go             - Comment queries
    dependencies.go         - Dependency queries
    features.go             - Feature queries
    roles.go, skills.go, dockerfiles.go, etc.
    sse.go                  - Server-Sent Events endpoint
    model_stats.go          - Model usage statistics
    cold_start_stats.go     - Cold start statistics
    timeline.go             - Task timeline
    tool_usage.go           - Tool usage queries

outbound/
  pg/                       - PostgreSQL persistence (pgx/v5 + pgxpool)
    pg.go                   - Main repository implementation (all SQL)
    pg_skills.go            - Skills SQL
    pg_dockerfiles.go       - Dockerfiles SQL
    migrations/             - SQL migration files (001_schema.sql, 002_features.sql)
    pg_test.go              - Integration tests (testcontainers + postgres:17)

init.go                     - Module initialization + DI wiring
```

### `internal/identity/` — Authentication bounded context
```
domain/
  types.go                  - User, Team, APIKey, Actor, MemberRole domain types
  repositories/users/       - User repository interface
  repositories/apikeys/     - API key repository interface
  repositories/teams/       - Team repository interface
  service/auth.go           - Auth service interface
app/
  auth.go                   - Auth logic (login, password, JWT, API keys)
  sso.go                    - SSO integration (Google, GitHub)
  teams.go                  - Team management
inbound/commands/           - HTTP handlers (login, register, refresh, API keys, SSO, teams, profile)
outbound/pg/                - PostgreSQL persistence + migrations
svrconfig/                  - Identity server config
```

### `internal/agach/` — TUI Monitor bounded context
```
domain/types.go             - TUI domain types
app/
  app.go                    - TUI application logic
  setup.go                  - Server setup wizard
  diagnostic.go             - Server diagnostic checks
inbound/tui/
  monitor.go                - Main TUI monitor view (bubbletea)
  welcome.go                - Welcome/setup screen
  diagnostic.go             - Diagnostic display
  sync.go                   - Data sync logic
  config.go                 - TUI configuration
  messages.go               - TUI message types
  tui.go                    - TUI initialization
  tcellapp/                 - tcell rendering
```

### Shared Packages (`pkg/`)
```
pkg/server/
  types.go                  - Shared API types (request/response structs)
  client/client.go          - Go client for the server HTTP API
pkg/middleware/
  middleware.go             - HTTP middleware (auth via JWT/API key, CORS, logging)
pkg/controller/
  controller.go             - Base HTTP controller helpers
pkg/apierror/
  apierror.go               - Standard API error type
pkg/agachconfig/
  config.go                 - Shared config (server URL, auth)
pkg/sse/
  hub.go                    - Server-Sent Events hub
pkg/websocket/
  hub.go                    - WebSocket hub for real-time updates
```
