---
name: doc-architecture
description: "Agach project hexagonal architecture: bounded contexts, source code hierarchy, entry points, layers, and package structure"
user-invocable: true
disable-model-invocation: false
---

# Agach Architecture — Hexagonal / Clean Architecture

## Entry Points (`cmd/`)
```
cmd/agach-server/
  main.go                         - HTTP + MCP server (port 8322, gorilla/mux, JWT auth, embedded React SPA)
  config.go                       - YAML config loader (SSO settings, DaemonJWTTTL)
cmd/agach-daemon/main.go          - Background daemon agent (Docker builds, chat sessions, git worktrees)
cmd/qa-seed/main.go               - Seeds database with deterministic test data, outputs JSON IDs for Playwright
```

## Backend — Hexagonal Architecture (`internal/`)

Each bounded context follows: `domain/` → `app/` → `inbound/` + `outbound/`

### `internal/server/` — Core server bounded context
```
domain/
  types.go                  - 11 ID types (ProjectID, TaskID, ColumnID, CommentID, FeatureID, SkillID,
                              DockerfileID, NotificationID, ChatSessionID, AgentID/RoleID, SpecializedAgentID)
                            - Enums: Priority, ColumnSlug, FeatureStatus, AuthorType,
                              NotificationSeverity, NotificationScope
                            - Structs: Project, Feature, Agent/Role, Skill, SpecializedAgent, Dockerfile,
                              Column, Task, Comment, TaskDependency, Notification, ChatSession,
                              ProjectSummary, ProjectWithSummary, FeatureWithTaskSummary, ProjectInfo,
                              TaskWithDetails, TokenUsage, ToolUsageStat, DependencyContext,
                              AgentColdStartStat, TimelineEntry, ModelTokenStat, ModelPricing, FeatureStats
  errors.go                 - 60+ domain errors (ErrNotFound, ErrInvalidMove, etc.) using domainerror.Error
  service/
    commands.go             - Command service interface (write operations) — composite of 8 sub-interfaces:
                              ProjectCommands, TaskCommands, AgentCommands, CommentCommands,
                              DependencyCommands, SkillCommands, FeatureCommands, DockerfileCommands,
                              NotificationCommands, SpecializedAgentCommands + top-level methods
    queries.go              - Query service interface (read operations) — composite of 8 sub-interfaces:
                              ProjectQueries, TaskQueries, AgentQueries, ColumnQueries, CommentQueries,
                              DependencyQueries, SkillQueries, FeatureQueries, DockerfileQueries,
                              NotificationQueries, StatsQueries, SpecializedAgentQueries
    servicetest/contract.go - Service mock + contract tests
  repositories/             - Repository interfaces + test contracts
    tasks/                  - TaskRepository (CRUD, GetNextTask, move, block, BulkCreate, BulkReassign, GetModelTokenStats)
    columns/                - ColumnRepository (FindByID, FindBySlug, List, EnsureBacklog)
    projects/               - ProjectRepository (CRUD, GetTree, GetSummary, CountChildren, ListModelPricing)
    comments/               - CommentRepository (CRUD, Count, IsLastComment)
    dependencies/           - DependencyRepository (CRUD, WouldCreateCycle, ListDependents, GetDependencyContext)
    features/               - FeatureRepository (CRUD, UpdateStatus, GetStats)
    agents/                 - AgentRepository (composite: GlobalAgent + ProjectAgent + ScopedAgent repos)
    skills/                 - SkillRepository (CRUD, IsInUse, ListByAgent, AssignToAgent, RemoveFromAgent)
    dockerfiles/            - DockerfileRepository (CRUD, IsInUse, SetLatest, Get/Set/ClearProjectDockerfile)
    notifications/          - NotificationRepository (CRUD, UnreadCount, MarkRead, MarkAllRead)
    toolusage/              - ToolUsageRepository (IncrementToolUsage, ListToolUsage)
    specialized/            - SpecializedAgentRepository (CRUD, ListByParent, ListSkills, SetSkills)
    chats/                  - ChatSessionRepository (CRUD, FindByFeature, UpdateState, UpdateJSONLPath, UpdateTokenUsage)
    Each has: <name>.go (interface) + <name>test/contract.go (mock + contract)

app/
  app.go                    - Application service (wires 13 service structs via Config)
  tasks.go                  - Task operations (create, bulk_create, move, block, complete, get_next, etc.)
  columns.go                - Column operations
  comments.go               - Comment operations
  dependencies.go           - Dependency operations (add, remove, cycle detection)
  roles.go                  - Agent CRUD + clone + project assignment + bulk reassign
  skills.go                 - Skill CRUD + agent assignment
  dockerfiles.go            - Dockerfile management + project dockerfile
  prompt.go                 - Prompt template generation
  agent_management.go       - Agent lifecycle management
  task_move_project.go      - Cross-project task moves
  tool_usage.go             - Tool usage tracking
  notifications.go          - Notification CRUD + mark read
  *_test.go                 - Unit tests using mock repositories

inbound/
  commands/                 - HTTP command handlers (POST/PUT/DELETE via gorilla/mux)
    router.go               - Route registration (13 handler groups)
    tasks.go, comments.go, projects.go, roles.go, skills.go, features.go,
    dockerfiles.go, project_agents.go, specialized_agents.go, notifications.go,
    seen.go, images.go, chats.go
  converters/               - Request/response type converters (API ↔ domain)
    tasks.go, columns.go, comments.go, projects.go, roles.go, features.go,
    skills.go, dockerfiles.go, notifications.go, timeline.go, tool_usage.go,
    specialized_agents.go, cold_start_stats.go, chats.go, mapslice.go
  queries/                  - HTTP query handlers (GET via gorilla/mux)
    router.go               - Query route registration (17 handler groups)
    tasks.go, projects.go, comments.go, dependencies.go, features.go,
    roles.go, skills.go, dockerfiles.go, notifications.go, project_agents.go,
    specialized_agents.go, sse.go, model_stats.go, cold_start_stats.go,
    timeline.go, tool_usage.go, chats.go

outbound/
  pg/                       - PostgreSQL persistence (pgx/v5 + pgxpool)
    pg.go                   - Main repository implementation
    pg_skills.go            - Skills SQL
    pg_dockerfiles.go       - Dockerfiles SQL
    pg_notifications.go     - Notifications SQL
    pg_test.go              - Integration tests (testcontainers + postgres:17)
    migrations/             - SQL migration files (001-006)

qaseed/seed.go              - QA test data seeder
init.go                     - Module initialization + DI wiring
```

### `internal/identity/` — Authentication bounded context
```
domain/
  types.go                  - User, Team, Actor, DaemonActor, MemberRole (admin/member)
  node.go                   - Node, OnboardingCode, NodeAccess, NodeID, NodeMode, NodeStatus
  errors.go                 - ErrUnauthorized, ErrForbidden, ErrInvalidCredentials, ErrSSOUserNoPassword, etc.
  ssoconfig.go              - SsoProvider, OIDCConfig, SAMLConfig, SsoConfig
  ttl.go                    - DefaultRefreshTokenTTL (7d), DefaultRememberMeTokenTTL (30d), DefaultDaemonJWTTTL (30d)
  repositories/users/       - UserRepository (CRUD, FindByEmail, FindBySSO, ListByTeam)
  repositories/teams/       - TeamRepository (CRUD, FindBySlug)
  repositories/nodes/       - NodeRepository (CRUD, ListByOwner, ListActiveByOwner, UpdateLastSeen)
  repositories/onboardingcodes/ - OnboardingCodeRepository (Create, FindByCode, MarkUsed, DeleteExpired)
  repositories/nodeaccess/  - NodeAccessRepository (GrantUser/Team, RevokeUser/Team, ListByNode, HasAccess)
  service/auth.go           - AuthCommands + AuthQueries interfaces
  service/teams.go          - TeamCommands + TeamQueries interfaces
  service/onboarding.go     - OnboardingCommands interface
  service/nodes.go          - NodeCommands + NodeQueries interfaces
app/
  auth.go                   - Auth logic (register, login, JWT, bcrypt cost 12, 15min access / 7-30d refresh)
  sso.go                    - SSO/OIDC integration (discovery, code exchange, JWK validation)
  teams.go                  - Team management (admin-only mutations)
  onboarding.go             - 6-digit code generation (15min expiry), node creation with refresh token
  nodes.go                  - Node management (revoke, rename, access grants)
  tokens.go                 - JWT issuance helpers (user + daemon tokens)
inbound/commands/
  auth.go                   - Login/register/refresh/logout/profile (rate limited: 5/15min per IP)
  sso.go                    - SSO providers list, OIDC authorize/callback with HMAC state
  teams.go                  - Team CRUD, user role/team assignment
  onboarding.go             - Generate code, complete onboarding, daemon token refresh
  nodes.go                  - Node list/get/revoke/rename/access
  actor.go                  - ActorFromRequest helper (Bearer token → Actor)
outbound/pg/                - PostgreSQL persistence + migrations (001-002)
                            - pgp_sym_encrypt for sensitive columns
init.go                     - DI wiring, default admin seeding (admin@agach.local/admin)
```

### `internal/daemon/` — Daemon agent bounded context
```
domain/
  types.go                  - OnboardingResult, ProjectInfo, DaemonState, BuildID, BuildStatus, DockerBuild
  ports.go                  - ServerAuth, ServerOnboarding, ServerConnection, ProjectFetcher, ChatUploader interfaces
config/config.go            - YAML config (.agach-daemon.yml), env vars (AGACH_ONBOARDING_CODE, AGACH_SERVER_URL)
app/
  app.go                    - Main daemon loop (onboarding → WS connect → event handling)
  tokens.go                 - Token persistence (~/.config/agach-daemon/tokens.json, 0600 perms)
  chat.go                   - ChatManager: spawns Claude CLI processes, captures JSONL, uploads to server
                              30min TTL with 25min warning, stats broadcasting every 5s
  docker.go                 - DockerService: list/rebuild/prune Docker images, SQLite build tracking
  git.go                    - GitService: clone/fetch repos, SSH/HTTPS auth, worktree management
client/
  auth.go                   - POST /api/daemon/refresh
  onboarding.go             - POST /api/onboarding/complete
  projects.go               - GET /api/projects/{id}
  websocket.go              - WS client with exponential backoff reconnect (1s-30s)
  chat_upload.go            - POST multipart JSONL upload
inbound/ws/
  hub.go                    - Local WS hub for TUI clients
  handlers.go               - Docker list/rebuild/logs/prune handlers
  types.go                  - HandlerFunc type
outbound/sqlite/
  sqlite.go                 - SQLite DB + migrations
  builds.go                 - DockerBuild repository (CRUD, DeleteNonLatest)
init.go                     - Daemon bootstrap (config → SQLite → app → run)
```

### Shared Infrastructure (`internal/pkg/`)
```
internal/pkg/apierror/
  apierror.go               - API error wrapper (Code, Message, Err)
internal/pkg/controller/
  controller.go             - HTTP response helpers (SendSuccess/SendFail/SendError, DecodeAndValidate)
                            - Custom validators: entity_id (UUID), slug (lowercase alphanum+hyphens)
internal/pkg/middleware/
  middleware.go             - RequestLogger, RequireAuth (JWT Bearer), LimitBodySize (512KB),
                              RateLimit (5/s per IP, 10 burst, 10min cleanup)
internal/pkg/websocket/
  hub.go                    - WebSocket hub (max 1000 clients, 64KB read, project-scoped broadcast)
  pump.go                   - Generic write pump
  constants.go              - WriteWait(10s), PongWait(60s), PingPeriod(54s), MaxMessageSize(4KB)
internal/pkg/sse/
  hub.go                    - Server-Sent Events hub (max 1000 subscribers/project, 1s heartbeat)
internal/agachconfig/
  config.go                 - Client config (.agach.yml), secure YAML loading (perms ≤ 0600)
```

### Public Packages (`pkg/`)
```
pkg/server/
  types.go                  - All HTTP request/response structs with validation tags
  client/client.go          - Go HTTP client SDK (projects, tasks, comments, dependencies, agents)
pkg/daemonws/
  types.go                  - Docker WebSocket message protocol (list, rebuild, logs, prune, events)
  chat.go                   - Chat WebSocket protocol (start, message, end, stats, TTL warning)
pkg/domainerror/
  error.go                  - Domain error type (Code + Message + wrapped error)
```
