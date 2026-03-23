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
cmd/agach-server/main.go    - HTTP + MCP server entry point
cmd/qa-seed/main.go         - Seeds database with test data for QA
```

## Backend — Hexagonal Architecture (`internal/`)

Each bounded context follows: `domain/` → `app/` → `inbound/` + `outbound/`

### `internal/kanban/` — Core Kanban bounded context
```
domain/
  types.go                  - Task, Column, Comment, Project, Role domain types
  errors.go                 - Domain errors (ErrNotFound, ErrInvalidMove, etc.)
  service/
    commands.go             - Command service interface (write operations)
    queries.go              - Query service interface (read operations)
    servicetest/contract.go - Service mock + contract tests
  repositories/             - Repository interfaces + test contracts
    tasks/                  - Task repository (CRUD, GetNextTask, move, block)
    columns/                - Column repository (fixed 4-column structure)
    projects/               - Project repository (hierarchical projects)
    comments/               - Comment repository
    dependencies/           - Task dependency repository
    features/               - Feature repository
    agents/                 - Agent management repository
    skills/                 - Skill repository
    dockerfiles/            - Dockerfile repository
    toolusage/              - Tool usage tracking repository
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
    tasks.go, columns.go, comments.go, projects.go, roles.go, etc.
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

outbound/
  pg/                       - SQLite persistence (named "pg" historically)
    pg.go                   - Main repository implementation (~70KB, all SQL)
    pg_skills.go            - Skills SQL
    pg_dockerfiles.go       - Dockerfiles SQL
    migrations/             - SQL migration files
    pg_test.go              - Integration tests

init.go                     - Module initialization + DI wiring
```

### `internal/identity/` — Authentication bounded context
```
domain/
  repositories/users/       - User repository interface
  repositories/apikeys/     - API key repository interface
  repositories/teams/       - Team repository interface
  service/auth.go           - Auth service interface
app/
  auth.go                   - Auth logic (login, password, API keys)
  sso.go                    - SSO integration
  teams.go                  - Team management
inbound/commands/           - HTTP handlers (login, register, API keys, teams)
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
pkg/kanban/
  types.go                  - Shared API types (request/response structs)
  client/client.go          - Go client for the kanban HTTP API
pkg/middleware/
  middleware.go             - HTTP middleware (auth, CORS, logging)
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
