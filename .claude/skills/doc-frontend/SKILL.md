---
name: doc-frontend
description: "Agach frontend: React/TypeScript pages, components, kanban UI, auth context, API client, WebSocket, hooks, modals"
user-invocable: true
disable-model-invocation: false
---

# Agach Frontend (`internal/server/ux/`)

## Structure
```
internal/server/ux/src/
  main.tsx                    - React app entry point
  App.tsx                     - Route configuration with protected routes, auth checks

  lib/
    api.ts                    - HTTP client for all backend endpoints (token refresh with 401 retry dedup)
    auth.ts                   - Auth helpers (JWT in localStorage: agach_access_token)
    types.ts                  - TypeScript interfaces mirroring pkg/server/types.go
    ws.ts                     - WebSocket client (reconnect with exponential backoff, token auth in query string)
    utils.ts                  - formatDuration() utility

  hooks/
    useChat.ts                - Chat session management (messages, stats, tokens, cost, WebSocket events)
    useWebSocket.ts           - Generic WebSocket subscription hook
    useSeenTasks.ts           - Deprecated (server-side now)
    useImageUpload.ts         - Image upload with state management

  pages/ (22 pages)
    HomePage.tsx              - Dashboard: project list, creation dialog, WebSocket updates
    LoginPage.tsx             - Email/password auth form, remember me, theme-aware
    KanbanPage.tsx            - Main board: drag-and-drop columns, search, role filters, multi-select, context menus
    BacklogPage.tsx           - Backlog view with move-to-todo, feature filtering
    FeaturesPage.tsx          - Feature/sub-project management
    FeatureDetailPage.tsx     - Single feature detail view
    FeatureChatPage.tsx       - Chat interface for collaborative feature discussions
    StatisticsPage.tsx        - Analytics: tool usage, timeline, token stats, model pricing
    ProjectSettingsPage.tsx   - Project configuration and agent management
    RolesPage.tsx             - Global/per-project agent management
    SkillsPage.tsx            - Skill management
    DockerfilesPage.tsx       - Docker template management
    AccountPage.tsx           - User profile, password management
    ApiKeysPage.tsx           - API key generation/management
    NotificationsPage.tsx     - Notification center with filtering
    NodesPage.tsx             - Node/daemon registration and management
    NodeSettingsPage.tsx      - Individual node settings
    OnboardingPage.tsx        - Onboarding code generation
    ExportGeminiPage.tsx      - Export project to Gemini format
    ExportClaudePage.tsx      - Export project to Claude format
    SpecializedAgentDetailPage.tsx - Specialized agent configuration
    SubProjectsPage.tsx       - Sub-project listing

  components/
    Layout.tsx                - Navigation sidebar, project context, features list, user menu, theme toggle, mobile responsive
    AuthContext.tsx            - Auth state management (login/logout, token persistence, 401 handling)
    ThemeContext.tsx           - Light/dark theme management, localStorage persistence
    NotificationBell.tsx      - Notification bell with unread count
    AgentSkillsPanel.tsx      - Display and manage agent skills

    kanban/
      Column.tsx              - Individual kanban column with task list
      TaskCard.tsx            - Task card: priority badge, status, role color
      TaskDrawer.tsx          - Side panel: full task details, comments, dependencies, edit (62.9K)
      TaskContextMenu.tsx     - Right-click: priority, role assignment, duplicate, move
      TaskActions.tsx         - Action modals: block, unblock, complete, wont-do, delete
      NewTaskModal.tsx        - Create task form with feature assignment (17.3K)
      CommentSection.tsx      - Task comments with edit capability (10.3K)
      FeatureDetailDrawer.tsx - Feature details and related tasks (10.3K)
      FeatureCard.tsx         - Feature card display
      BlockedBanner.tsx       - Blocked task information banner
      BulkActionsBar.tsx      - Multi-select task actions bar

    modals/ (8 task modals)
      BlockTaskModal.tsx      - Block task with reason
      UnblockTaskModal.tsx    - Unblock task
      CompleteTaskModal.tsx   - Mark done with completion summary
      DeleteTaskModal.tsx     - Delete confirmation
      MarkWontDoModal.tsx     - Request won't-do with reason
      ApproveWontDoModal.tsx  - Approve won't-do
      CommentWontDoModal.tsx  - Comment on won't-do request
      MoveToProjectModal.tsx  - Move task to different project

    dialogs/
      CreateProjectDialog.tsx      - Project creation form
      AddAgentToProjectDialog.tsx  - Assign agents to projects
      RemoveAgentDialog.tsx        - Remove agents from projects
      CloneAgentDialog.tsx         - Clone agent with new name
      EditSpecializedAgentDialog.tsx - Edit specialized agent
      DeleteConfirmModal.tsx       - Generic confirmation modal

    settings/                 - Settings layout components
    ui/                       - Reusable UI primitives
```

## Key Patterns
- `data-qa` attributes on interactive elements for Playwright tests
- JSend response wrapper (status/data/error)
- Task drawer URL-driven via searchParams `?task=`
- Search debouncing (300ms)
- Multi-select with Ctrl+click, bulk actions bar
- Role-based UI with color coding
- WebSocket event types: task_created, task_updated, task_deleted, task_moved, task_completed,
  project_created/updated/deleted, comment_added/edited, agent/skill/feature CRUD, notification,
  wip_slot_available, wont_do_rejected, task_wont_do
