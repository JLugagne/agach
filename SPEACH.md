<p align="center">
  <img src="images/logo.svg" width="200px" alt="Agach" />
</p>

<h3 align="center">Describe the feature. Agents build it.</h3>

<p align="center">
  Agach is the orchestration layer between your team and AI coding agents.<br/>
  A PM describes what to build. A planning agent refines the spec.<br/>
  Autonomous workers execute it on your codebase.<br/>
  <strong>Open source. Self-hosted. Your code never leaves your infrastructure.</strong>
</p>

<p align="center">
  <a href="#how-it-works">How It Works</a> ·
  <a href="#why-agach">Why Agach</a> ·
  <a href="#quick-start">Quick Start</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#features">Features</a>
</p>

---

## The problem

AI can write code. Nobody is managing the work.

Every agent session starts cold — prior decisions, architectural choices, what was tried and failed, all gone. Specs are a prompt you type and hope for the best. When a feature breaks six weeks later, the fixing agent has no idea what the original intent was, what was deliberately excluded, or which contracts must still hold. Meanwhile your PM is asking "what's the status?" and the answer lives in twelve terminal windows nobody is watching.

The tools exist to write code with AI. What's missing is the system that coordinates the work — from feature definition through execution to bug resolution — with full traceability, context preservation, and visibility for everyone on the team, not just the person in the terminal.

## How it works

```
                    ┌──────────────────────────────────────────────┐
                    │              Agach Server                    │
                    │                                              │
 ┌──────────┐      │  ┌──────────────┐    ┌───────────────────┐   │
 │    PM    ├──────►│  │ Planning Chat │    │  Dashboard        │   │
 │          │◄──────┤  │ (negotiate    │    │  (track progress, │   │
 └──────────┘      │  │  scope, specs)│    │   costs, status)  │   │
                    │  └──────┬───────┘    └───────────────────┘   │
                    │         │ task files + acceptance criteria    │
                    │         ▼                                    │
                    │  ┌──────────────┐    ┌───────────────────┐   │
                    │  │   Features   │    │  Conversations    │   │
                    │  │   Tasks      │    │  (stored, search- │   │
                    │  │   Files      │    │   able, resumable)│   │
                    │  └──────┬───────┘    └───────────────────┘   │
                    └─────────┼────────────────────────────────────┘
                              │ REST API
                              ▼
                    ┌──────────────────────────────────────────────┐
                    │              Agach Daemon                    │
                    │                                              │
                    │  ┌─────────────────────────────────────────┐ │
                    │  │ Orchestrator                            │ │
                    │  │  - pulls unblocked tasks                │ │
                    │  │  - creates git worktrees                │ │
                    │  │  - spawns Claude Code sessions          │ │
                    │  │  - monitors summary files               │ │
                    │  │  - pushes status back to server         │ │
                    │  │  - creates merge requests               │ │
                    │  └────────┬──────────────┬─────────────────┘ │
                    │           │              │                   │
                    │    ┌──────▼──────┐ ┌────▼────────┐          │
                    │    │  Worktree   │ │  Worktree   │  ...     │
                    │    │  ┌───────┐  │ │  ┌───────┐  │          │
                    │    │  │ Agent │  │ │  │ Agent │  │          │
                    │    │  │ (T3)  │  │ │  │ (T4)  │  │          │
                    │    │  └───────┘  │ │  └───────┘  │          │
                    │    └─────────────┘ └─────────────┘          │
                    │           │              │          MCP      │
                    │    ┌──────▼──────────────▼─────────────────┐ │
                    │    │ MCP Tools: create_subtask,            │ │
                    │    │ complete_task, report_blocked,         │ │
                    │    │ attach_file                            │ │
                    │    └──────────────────────────────────────┘ │
                    └──────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Your pipeline   │
                    │  GitHub / GitLab  │
                    │  CodeRabbit       │
                    │  CI/CD            │
                    └──────────────────┘
```

### Step 1 — Describe what you want

Open the dashboard. Create a feature. A planning agent joins the chat and starts asking the right questions — scope, edge cases, acceptance criteria. You go back and forth until the spec is solid. Close the tab, come back tomorrow, pick up where you left off. The full conversation is stored and searchable.

```
PM:     We need OAuth2 login with Google
Agent:  Should we support refresh tokens and silent re-auth?
PM:     Yes, and revocation on logout
Agent:  Got it. I'll add a task for token cleanup on session end.
        Creating 6 tasks with dependencies...
```

The planning agent produces structured task files with UUIDs, dependencies, role assignments, acceptance criteria, and any files you uploaded (mockups, specs, CSVs). These go into agach as the source of truth.

### Step 2 — Agents execute the work

The daemon watches for unblocked tasks. When one appears, it:

1. Creates an isolated git worktree
2. Drops the task file in, along with parent summary files
3. Spawns a Claude Code session scoped to that task
4. Waits for the agent to write its summary file
5. Pushes status back to agach and checks what's unblocked next

Each agent reads only what it needs — its own task file and the summaries from parent tasks. No context bloat. No cross-contamination between parallel work streams.

```
[14:02] picking up T3 — refresh token rotation
        creating worktree at /tmp/agach/feat-oauth/T3
        loading parent summaries: T1, T2
        spawning claude code session...

[14:18] T3 completed — summary written
        pushing status to agach
        tokens: 42,180 in / 8,340 out

[14:18] T4 unblocked — token revocation on logout
        creating worktree...
```

### Step 3 — Review and merge

When all tasks pass, agach creates a merge request. Your existing review tools handle the rest — CodeRabbit, Graphite, GitHub reviews, your CI pipeline. If review comments come back, the daemon can spawn an agent to address them.

Agach doesn't replace your pipeline. It feeds it.

### Step 4 — Bugs flow backward

This is where agach is different from everything else.

A bug report against a completed feature triggers context regeneration. The original task files are rebuilt with acceptance criteria marked green or red. The fixing agent reads the original planning conversation, the parent summaries, and sees exactly which contract broke:

```
✓  Provider config loads from env
✓  Login redirects to Google
✗  Refresh token rotates on expiry       ← broken
✓  Logout revokes all tokens
✓  Integration tests pass
```

No more "here's a vague ticket, go figure it out." The agent knows what was built, why it was built that way, what was deliberately excluded, and which criteria must still hold after the fix.

## Why agach

### vs. single-agent tools (Claude Code, Cursor, Copilot)

These are excellent at writing code. They have no idea what to write, no memory between sessions, and no way for non-technical stakeholders to see what's happening. Agach adds the orchestration layer: specification through conversation, context preservation through summary chains, and visibility through the dashboard.

### vs. multi-agent orchestrators (Gas Town, Agent Orchestrator, Multiclaude)

These solve parallelism — running many agents at once. They're built for developers managing terminal windows. Agach starts earlier (the planning chat) and goes further (regression-aware bug fixing). The dashboard is built for PMs and leads, not just engineers. And agach is cost-conscious by design: sequential execution with parallel only when file sets don't overlap, rather than swarming 30 agents and burning $100/hour.

### vs. AI app generators (Lovable, Bolt, v0)

These generate apps from scratch. Agach works on existing codebases with existing teams and existing CI pipelines. Lovable replaces the first sprint. Agach replaces the engineering manager.

## Features

### Planning chat

A structured conversation between your PM and an AI planning agent. Negotiates scope, flags risks, defines acceptance criteria, and produces machine-readable task files. Conversations are stored, searchable, and resumable across sessions. The planning chat is the "why" behind every feature — six months later, it's still there.

### Summary chains

Each agent writes a structured summary on completion: changes made, key decisions, files modified, breaking changes. Downstream agents read their parents' summaries. Information compresses naturally — each link in the chain carries forward only what's relevant. No context bloat, no 50MB prompt.

### Regression-aware bug fixing

Bug reports against completed features regenerate the full context — original task files, acceptance criteria, planning conversation — with the broken criterion marked red. The fixing agent sees the original intent, not just the current code.

### File attachments

Specs, mockups, screenshots, CSVs — attach them to features and tasks. When the daemon sets up a worktree, attached files land on disk next to the task file. PMs share context the way they naturally work. Agents receive it without prompt engineering.

### Dashboard

Progress per feature. Token costs per task. Which agent is working on what. Links to merge requests, CI runs, and reviews. Non-technical stakeholders see everything without opening a terminal.

### Daemon orchestrator

Pulls unblocked tasks from agach's API. Creates git worktrees. Spawns isolated Claude Code sessions. Monitors for summary files. Pushes status. Creates merge requests. Runs on your laptop, a build server, or in CI. Multiple daemons can connect to the same agach instance.

### Integrates, doesn't replace

GitHub for git. CodeRabbit for code review. Your CI for validation. Linear or Jira for upstream tracking. Agach orchestrates the flow between tools you already trust. Nothing to rip out.

## Quick start

### 1. Run the server

```bash
git clone https://github.com/JLugagne/agach-mcp.git
cd agach-mcp
docker-compose up --build
```

Open `http://localhost:8322` — you'll see the dashboard.

### 2. Start the daemon

```bash
agach daemon start --server http://localhost:8322
```

The daemon connects to agach and watches for unblocked tasks.

### 3. Create a feature

Open the dashboard, click "New Feature," and start chatting with the planning agent. Describe what you want. Negotiate the spec. When you're satisfied, the planning agent creates tasks and the daemon picks them up automatically.

## Architecture

Agach is two components with a clean separation:

**Server** — a web application with a REST API, WebSocket for real-time updates, the dashboard UI, and the planning chat. It stores features, tasks, conversations, and files. Humans interact with it through the browser. It has zero knowledge of Claude Code, agents, worktrees, or MCP.

**Daemon** — the orchestration engine. It talks to the server's REST API to read tasks and push status updates. It manages worktrees, spawns Claude Code sessions, monitors for summary files, handles the dependency graph, and creates merge requests. The MCP tools live here — when a sub-agent needs to create a subtask or report completion, it calls MCP tools exposed by the daemon.

```
cmd/
  agach-server/         # Server entry point
  agach/                # Daemon + TUI entry point
internal/
  kanban/
    domain/             # Types, errors, repository interfaces
    app/                # Business logic (commands & queries)
    inbound/
      commands/         # REST write endpoints
      queries/          # REST read endpoints
      converters/       # Domain ↔ public type mapping
    outbound/
      sqlite/           # SQLite repositories + migrations
    init.go             # Dependency injection
  agach/
    app/                # Daemon orchestration logic
    inbound/tui/        # Terminal UI
pkg/
  kanban/               # Public types with validation
  controller/           # HTTP response helpers
  websocket/            # WebSocket hub
  sse/                  # Server-sent events hub
ux/                     # React + TypeScript + Tailwind (dashboard)
```

### Tech stack

| Layer | Technology |
|-------|------------|
| Backend | Go |
| Storage | SQLite (WAL mode, per-project databases) |
| Frontend | React, TypeScript, Tailwind CSS, Vite |
| Real-time | WebSocket |
| Agent protocol | MCP over stdio (daemon ↔ agents) |
| Server ↔ Daemon | REST API |

### Design principles

**The server is a project management tool.** It doesn't know about agents or code. You could use it without the daemon as a planning tool with the chat interface.

**The daemon is an optional execution layer.** It adds autonomous work to the server. Multiple daemons can connect to the same server instance.

**Files over protocol for agent context.** Agents read task files and summary files from disk. MCP is only used for writes (create subtask, report completion). This minimizes token overhead and maximizes context quality.

**Sequential by default, parallel when safe.** The daemon serializes tasks unless their file sets are provably disjoint. This avoids merge conflicts and keeps quality high. Speed comes from eliminating human coordination overhead, not from swarming agents.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AGACH_HOST` | `127.0.0.1` | Server bind address |
| `AGACH_PORT` | `8322` | Server port (dashboard + REST API + WebSocket) |
| `AGACH_DATA_DIR` | `./data` | SQLite databases directory |

## Status

Agach is under active development. The kanban server, dashboard, and TUI are working. The planning chat, daemon orchestrator, and regression system are in progress.

If you're interested in this approach to AI-assisted development, star the repo and watch for updates. Contributions welcome.

## License

MIT
