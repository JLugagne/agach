---
name: doc-daemon
description: "Agach daemon agent: onboarding, WebSocket reconnect, Docker builds, chat sessions (Claude CLI), git worktrees, SQLite persistence"
user-invocable: true
disable-model-invocation: false
---

# Agach Daemon (`internal/daemon/`)

## Overview
Background agent that onboards with the server, maintains a WebSocket connection, manages Docker image builds, spawns Claude CLI chat sessions, and manages git worktrees.

## Domain Types (`domain/`)

### types.go
- `OnboardingResult` — AccessToken, RefreshToken, NodeID, NodeName, Mode
- `ProjectInfo` — ID, Name, GitURL
- `DaemonState` — StateDisconnected, StateOnboarding, StateConnected
- `BuildID`, `BuildStatus` — pending, building, success, failed
- `DockerBuild` — ID, DockerfileSlug, Version, ImageHash, ImageSize, Status, BuildLog, timestamps

### ports.go (interfaces)
- `ServerAuth` — RefreshDaemonToken(nodeID, refreshToken) → accessToken
- `ServerOnboarding` — CompleteOnboarding(code, nodeName) → OnboardingResult
- `ServerConnection` — RunWithReconnect(ctx), Send(msg)
- `ProjectFetcher` — GetProject(token, projectID) → ProjectInfo
- `ChatUploader` — UploadJSONL(token, projectID, featureID, sessionID, filePath)

## Configuration (`config/config.go`)
- File: `~/.config/agach/daemon.yml` (or AGACH_SERVER_URL env var)
- Fields: base_url, node_name
- Env-only: AGACH_ONBOARDING_CODE (6-digit code)
- Derived: SQLitePath() → daemon.db, WebSocketURL() → ws:// or wss://

## App Layer (`app/`)

### Main Loop (`app.go`)
States: Init → Onboarding → Connected → Reconnecting → Stopped
- `Run(ctx)` — onboard → connect WS → handle events until stopped
- WebSocket event routing: docker.list, docker.rebuild, docker.logs, docker.prune,
  chat.start, chat.user_msg, chat.end, chat.ping

### Token Store (`tokens.go`)
- Persists to `~/.config/agach-daemon/tokens.json` (0600 permissions)
- Fields: access_token, refresh_token, node_id, node_name

### Chat Manager (`chat.go`)
- Spawns Claude CLI with `--output-format stream-json`
- Captures stdout/stderr, writes JSONL log
- 30-minute idle TTL with 25-minute warning broadcast
- Stats broadcaster every 5 seconds (token counts, cost, duration)
- Session lifecycle: StartSession → SendMessage → EndSession (stop process, upload JSONL)
- `ChatSession` tracks: ID, FeatureID, ProjectID, ClaudeSessionID, WorktreePath,
  token counts, model, message count, timestamps

### Docker Service (`docker.go`)
- `ListImages(ctx)` — groups builds by dockerfile slug
- `Rebuild(ctx, slug, eventCh)` — creates build record, streams events
- `GetBuildLogs(ctx, buildID)` — returns build log
- `PruneNonLatest(ctx, eventCh)` — removes all but latest per slug

### Git Service (`git.go`)
- Cache dir: `~/.cache/agach`
- `EnsureWorktree(projectID, gitURL, mainBranch)` — clone or fetch+pull
- Auth: SSH keys or GITHUB_TOKEN for HTTPS

## Client Layer (`client/`)
- `AuthClient` — POST /api/daemon/refresh
- `OnboardingClient` — POST /api/onboarding/complete (error codes: CODE_NOT_FOUND, CODE_EXPIRED, CODE_ALREADY_USED)
- `ProjectClient` — GET /api/projects/{projectID}
- `WSClient` — WebSocket with exponential backoff reconnect (1s → 30s max)
- `ChatUploadClient` — POST multipart JSONL upload

## Inbound WebSocket Hub (`inbound/ws/`)
- Local hub for TUI clients connecting to the daemon
- Handlers: HandleListDockerfiles, HandleRebuild (async), HandleGetLogs, HandlePrune (async)
- Build/prune events forwarded to hub as they occur
- HandlerFunc signature: `func(ctx, daemonws.Message) (daemonws.Message, error)`

## Outbound SQLite (`outbound/sqlite/`)
- Database: daemon.db (local)
- `builds` table: id, dockerfile_slug, version, image_hash, image_size, status, build_log, timestamps
- Repository: Create, FindByID, ListByDockerfile, ListAll, UpdateStatus, Delete, DeleteNonLatest

## WebSocket Protocol (`pkg/daemonws/`)

### Docker Messages
- `docker.list` / `docker.rebuild` / `docker.logs` / `docker.prune`
- Events: `build.event` (slug, build_id, status, log), `prune.event` (slug, removed, total)

### Chat Messages
- `chat.start` → ChatStartRequest (session_id, feature_id, project_id, node_id, resume_session_id)
- `chat.user_msg` → ChatUserMessage (session_id, content)
- `chat.end` → EndSession
- `chat.ping` → RefreshActivity
- Events: `chat.message`, `chat.stats`, `chat.end`, `chat.error`, `chat.ttl_warning`

## Init (`init.go`)
1. Parse `-init` flag (create default config)
2. Load config from file or env
3. Create SQLite database + migrations
4. Create build repository
5. Instantiate daemon app
6. Signal handling (SIGINT, SIGTERM)
7. Run with reconnect logic
