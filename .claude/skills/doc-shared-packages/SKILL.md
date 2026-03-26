---
name: doc-shared-packages
description: "Agach shared infrastructure: internal/pkg (controller, middleware, websocket, sse, apierror), pkg (server types, client SDK, daemonws, domainerror), agachconfig"
user-invocable: true
disable-model-invocation: false
---

# Agach Shared Packages

## Public Packages (`pkg/`)

### pkg/server/types.go — HTTP Request/Response Types
All REST API request and response structs with `validate:` struct tags.
- Project: CreateProjectRequest, UpdateProjectRequest, ProjectResponse, ProjectSummaryResponse
- Agent: CreateAgentRequest, UpdateAgentRequest, AgentResponse, CloneAgentRequest
- Specialized: CreateSpecializedAgentRequest, UpdateSpecializedAgentRequest, SpecializedAgentResponse
- Skill: CreateSkillRequest, UpdateSkillRequest, SkillResponse, AddSkillToAgentRequest
- Task: CreateTaskRequest, UpdateTaskRequest, MoveTaskRequest, CompleteTaskRequest, BlockTaskRequest,
  RequestWontDoRequest, RejectWontDoRequest, ReorderTaskRequest, MoveTaskToProjectRequest, TaskResponse, TaskWithDetailsResponse
- Comment: CreateCommentRequest, UpdateCommentRequest, CommentResponse
- Board: ColumnResponse, BoardResponse, ColumnWithTasksResponse, DependencyContextResponse, AddDependencyRequest
- Feature: CreateFeatureRequest, UpdateFeatureRequest, UpdateFeatureStatusRequest, FeatureResponse, FeatureWithSummaryResponse
- Dockerfile: CreateDockerfileRequest, UpdateDockerfileRequest, DockerfileResponse, SetProjectDockerfileRequest
- Notification: CreateNotificationRequest, NotificationResponse
- Chat: StartChatSessionRequest, ChatSessionResponse
- Stats: ToolUsageStatResponse, TimelineEntryResponse, ColdStartStatResponse, TasksByAgentResponse
- Assignment: AssignAgentToProjectRequest, RemoveAgentFromProjectRequest, BulkReassignTasksRequest/Response
- Errors: ErrInvalidProjectRequest, ErrInvalidTaskRequest, etc. (using apierror.Error)

### pkg/server/client/client.go — HTTP REST Client SDK
- `New(baseURL)` — validates URL (http/https only, rejects link-local/metadata IPs)
- Max response: 10 MB, max session ID: 512 chars
- Methods: ListProjects, GetProject, CreateProject, ListProjectRoles, CreateProjectAgent,
  UpdateProjectAgent, DeleteProjectAgent, GetNextTasks, WaitForNextTask (SSE blocking),
  UpdateTaskSessionID, UpdateTask, ListTasks, CreateTask, CompleteTask, BlockTask,
  MoveTask, GetColumnCounts, AddComment, ListComments, GetColumns, AddDependency

### pkg/daemonws/ — Daemon WebSocket Protocol
- **types.go**: Message (type, request_id, payload, error), BuildEvent, PruneEvent
  Types: docker.list, docker.rebuild, docker.logs, docker.prune, build.event, prune.event, error
- **chat.go**: ChatStartRequest, ChatStartResponse, ChatUserMessage, ChatMessageEvent,
  ChatStatsEvent, ChatEndEvent, ChatErrorEvent, ChatTTLWarningEvent
  Types: chat.start, chat.message, chat.user_msg, chat.end, chat.error, chat.stats, chat.ping, chat.ttl_warning

### pkg/domainerror/error.go — Domain Error Type
- `Error` struct: Code (string), Message (string), wraps underlying error
- `IsDomainError(err) bool` — type assertion helper
- Implements: Error(), Unwrap(), ErrorCode(), ErrorMessage()

## Internal Shared Infrastructure (`internal/pkg/`)

### internal/pkg/apierror/apierror.go — API Error Wrapper
- `Error` struct: Code, Message, Err
- Distinct from domain errors; used by inbound layer for HTTP error responses
- Implements: Error(), Unwrap(), ErrorCode(), ErrorMessage()

### internal/pkg/controller/controller.go — HTTP Response Helpers
- `NewController(logger)` — registers custom validators: `entity_id` (UUID), `slug` (lowercase alphanum+hyphens, max 100)
- `SendSuccess(w, r, data)` — 200 OK with JSend wrapper `{status: "success", data: ...}`
- `SendFail(w, r, statusCode, err)` — 4xx with `{status: "fail", error: {code, message}}`
- `SendError(w, r, err)` — 500 with `{status: "error", error: {code, message}}`
- `DecodeAndValidate(r, dest, validationErr)` — JSON decode + struct tag validation, rejects non-application/json
- `CodedError` interface — allows errors to carry Code/Message without circular imports

### internal/pkg/middleware/middleware.go — HTTP Middleware
- `RequestLogger` — logs method, path, status, duration, remote IP
- `NewRequireAuth(authValidator)` — JWT Bearer validation, injects Actor into context (`ActorContextKey = "actor"`)
  Sets headers: X-Content-Type-Options, X-Frame-Options, Cache-Control, CORS
- `LimitBodySize` — 512 KB request body limit (413 if exceeded)
- `RateLimit` — 5 req/s per IP, 10 burst, 10min cleanup of old limiters (429 if exceeded)
  Uses real RemoteAddr only (never trusts X-Forwarded-For)

### internal/pkg/websocket/ — WebSocket Hub
- **hub.go**: Hub (max 1000 clients, 256 broadcast buffer), Client (conn, send channel, project_id, is_daemon, node_id)
  `Broadcast(Event)` — project-scoped delivery, `SendToDaemon(nodeID, data)` — targeted delivery
  `RegisterHandler(msgType, fn)` — message type routing, `NewRelayHandler()` — daemon↔client relay
  Read/write: 64KB read limit, 10s write deadline, 60s pong wait, 54s ping period
- **pump.go**: Generic `RunWritePump[M]()` and `HandleUnregister[C]()` helpers
- **constants.go**: WriteWait(10s), PongWait(60s), PingPeriod(54s), MaxMessageSize(4KB)

### internal/pkg/sse/hub.go — Server-Sent Events Hub
- Max 1000 subscribers per project
- 1-second heartbeat (":" keep-alive)
- `Subscribe(projectID)` → (chan string, unsubscribe func)
- `Publish(projectID, data)` — sanitizes newlines, evicts slow consumers
- `HasSubscribers(projectID) bool`

## Client Configuration (`internal/agachconfig/`)

### config.go — Daemon/Client Config
- `Config` struct: BaseURL (yaml: base_url)
- `FindConfigFile(dir, filename, maxDepth)` — walks up directory tree (max 5 levels)
- `LoadSecureYAML(path, dest)` — requires file permissions ≤ 0600
- `ValidateBaseURL(url)` — enforces https for remote hosts
- `Load(dir)` — looks for .agach.yml in dir and parents
