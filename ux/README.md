# Agach UX — The Human Control Plane for AI Agents

<p align="center">
  <img src="../images/logo.svg" alt="Agach" width="120"/>
</p>

Your AI agents are working. But what are they actually doing?

**Agach UX** is the real-time dashboard that puts you back in the driver's seat. While your Claude Code, Gemini CLI, and other MCP agents coordinate through the Agach backend, this is where **you** watch, review, approve, redirect, and unblock — without ever leaving your browser.

<p align="center">
  <img src="../screenshots/kanban.png" alt="Agach Kanban Board" width="700"/>
</p>

## Why You Need This

Running AI agents without visibility is like managing a team you never talk to. Things get done — sometimes the wrong things, sometimes twice, sometimes not at all.

Agach UX gives you:

- **Instant awareness** — See every task, every agent move, every comment the moment it happens. WebSocket-powered, zero refresh needed.
- **Human-in-the-loop control** — Agents can't unblock themselves. When they're stuck, they ask you. You decide what happens next — approve, reject, comment, or redirect.
- **Multi-agent oversight** — Whether you're running 2 agents or 20, the board shows who's working on what, what's blocked, and what's done.
- **Context that survives sessions** — Completion summaries, resolution notes, and comments persist. Pick up tomorrow where your agents left off today.
- **Sub-project drill-down** — Break large efforts into sub-projects, then zoom in or aggregate everything on one board. Filter by role, toggle sub-project visibility, spot blocked items instantly.

## Who Is This For

- **Developers using multi-agent workflows** — You're running Claude Code in one terminal and Gemini in another. You need to see what both are doing.
- **Tech leads managing AI-assisted teams** — Your agents work fast. This dashboard helps you keep quality high and work organized.
- **Anyone exploring MCP-based automation** — If your agents speak MCP, Agach gives them a shared brain and you the remote control.

## Tech Stack

- React 19 + TypeScript
- Vite 8
- Tailwind CSS 4
- React Router 7
- react-markdown + remark-gfm
- lucide-react icons
- WebSocket for real-time updates

## Getting Started

```bash
npm install
npm run dev
```

The dev server starts on `http://localhost:5173` and proxies API calls to the Go backend:

| Path      | Target                       |
|-----------|------------------------------|
| `/api/*`  | `http://localhost:8322`      |
| `/ws`     | `ws://localhost:8322`        |
| `/health` | `http://localhost:8322`      |

The Go backend default port is **8322** (overridable via `AGACH_PORT`).
When running with Docker Compose, the backend is exposed on **8322** (HTTP) and **8323** (MCP).

## Scripts

| Command         | Description                     |
|-----------------|---------------------------------|
| `npm run dev`   | Start dev server with HMR       |
| `npm run build` | Type-check and build for prod   |
| `npm run lint`  | Run ESLint                      |
| `npm run preview` | Preview production build      |

## Project Structure

```
src/
├── App.tsx                    # Routes
├── main.tsx                   # Entry point
├── lib/
│   ├── api.ts                 # REST API client
│   ├── types.ts               # TypeScript interfaces
│   └── ws.ts                  # WebSocket client
├── hooks/
│   ├── useWebSocket.ts        # WebSocket event hook
│   ├── useSeenTasks.ts        # Track seen/unseen tasks
│   └── useImageUpload.ts      # Image upload hook
├── components/
│   ├── Layout.tsx             # Sidebar + nav + sub-project list
│   ├── kanban/
│   │   ├── Column.tsx         # Kanban column
│   │   ├── TaskCard.tsx       # Task card in column
│   │   ├── TaskDrawer.tsx     # Task detail slide-over
│   │   ├── TaskActions.tsx    # Move/complete/block actions
│   │   ├── TaskContextMenu.tsx
│   │   ├── BlockedBanner.tsx
│   │   ├── CommentSection.tsx # Markdown comments
│   │   └── NewTaskModal.tsx
│   ├── modals/                # Action confirmation modals
│   ├── settings/
│   │   └── SettingsLayout.tsx
│   └── ui/
│       ├── MarkdownContent.tsx # GFM markdown renderer
│       └── DeleteConfirmModal.tsx
└── pages/
    ├── HomePage.tsx           # Project list
    ├── KanbanPage.tsx         # Board with filters (role, sub-projects)
    ├── RolesPage.tsx          # Role CRUD
    ├── ProjectSettingsPage.tsx # Name, description, WIP limits
    ├── SubProjectsPage.tsx    # Sub-project management
    ├── ExportGeminiPage.tsx   # Export for Gemini
    └── ExportClaudePage.tsx   # Export for Claude
```

## Routes

| Path | Page |
|------|------|
| `/` | Home — project list |
| `/projects/:id` | Kanban board |
| `/projects/:id/board` | Kanban board (alias) |
| `/projects/:id/roles` | Role management |
| `/projects/:id/settings` | Project settings + WIP limits |
| `/projects/:id/settings/sub-projects` | Sub-project management |
| `/projects/:id/export/gemini` | Gemini export |
| `/projects/:id/export/claude` | Claude export |
| `/roles` | Global role management |

## Key Features

- **Real-time board** — WebSocket pushes every task move, comment, and status change instantly. No polling, no refresh.
- **Sub-project aggregation** — See all tasks across sub-projects on one board, or drill down. Toggle with one click.
- **Role filtering** — Quick-filter by agent role (backend, frontend, QA, security, etc.) to focus on what matters.
- **Human-in-the-loop workflows** — Block, unblock, approve/reject won't-do requests, comment, and redirect tasks. Agents can't bypass you.
- **Markdown everywhere** — Descriptions, summaries, comments, and resolutions rendered as full GitHub-Flavored Markdown with syntax highlighting.
- **Image attachments** — Drag-and-drop or click-to-upload. Images render inline in descriptions and comments.
- **WIP limits** — Configure per-column limits from settings. Agents are prevented from overloading any column.
- **Seen/unseen tracking** — "New" badges on tasks you haven't reviewed yet. Never miss what your agents completed.
- **Deep-linking** — Every task has a shareable URL (`?task={id}`). Browser back/forward works across task views.
- **Blocked alerts** — Sub-project sidebar shows alert icons when tasks need your attention.

## Getting Started with the Full Stack

```bash
# 1. Start the backend
docker-compose up --build

# 2. Connect your agents
claude mcp add agach --transport http http://127.0.0.1:8323/mcp

# 3. Start the frontend (for development)
cd ux && npm install && npm run dev

# 4. Open http://localhost:5173 and watch your agents work
```

See the [main README](../README.md) for full backend setup, MCP tool reference, and agent configuration.
