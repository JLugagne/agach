---
name: doc-frontend
description: "Agach frontend: React/TypeScript pages, components, kanban UI, auth context, API client, WebSocket, Playwright tests, data-qa conventions"
user-invocable: true
disable-model-invocation: false
---

# Agach Frontend (`ux/`)

## Structure
```
ux/src/
  App.tsx                   - Root app with routing
  lib/
    api.ts                  - HTTP client for all backend endpoints
    auth.ts                 - Auth helpers (JWT tokens)
    types.ts                - TypeScript types mirroring pkg/kanban/types.go
    ws.ts                   - WebSocket client
    utils.ts                - Utility functions
  pages/
    KanbanPage.tsx           - Main kanban board view
    FeaturesPage.tsx         - Feature list page
    FeatureDetailPage.tsx    - Feature detail page
    BacklogPage.tsx          - Backlog management
    RolesPage.tsx            - Role management
    SkillsPage.tsx           - Skill management
    StatisticsPage.tsx       - Usage statistics
    ProjectSettingsPage.tsx  - Project settings
    SubProjectsPage.tsx      - Sub-project management
    DockerfilesPage.tsx      - Dockerfile management
    LoginPage.tsx            - Login page (authentication required)
    AccountPage.tsx          - Account settings
    ApiKeysPage.tsx          - API key management
    HomePage.tsx             - Home/project selection
    ExportClaudePage.tsx     - Export for Claude
    ExportGeminiPage.tsx     - Export for Gemini
  components/
    kanban/                  - Kanban-specific components
      Column.tsx             - Board column
      TaskCard.tsx           - Task card
      TaskDrawer.tsx         - Task detail drawer (main interaction point)
      NewTaskModal.tsx       - Task creation modal
      CommentSection.tsx     - Comment display/input
      FeatureCard.tsx        - Feature card
      FeatureDetailDrawer.tsx - Feature detail
      TaskActions.tsx        - Task action buttons
      TaskContextMenu.tsx    - Right-click context menu
      BulkActionsBar.tsx     - Bulk selection actions
      BlockedBanner.tsx      - Blocked task banner
    Layout.tsx               - App shell layout with sidebar
    AuthContext.tsx           - Auth context provider (JWT storage)
    ThemeContext.tsx          - Theme context provider
    modals/                  - Shared modal components
    settings/                - Settings layout components
    ui/                      - Reusable UI primitives
```

## Development Guidelines

### React Components
- Add `data-qa` attributes to interactive elements (buttons, inputs, modals, cards) to support Playwright tests
- Use descriptive values: `data-qa="task-card"`, `data-qa="create-task-btn"`, `data-qa="task-drawer"`

## Playwright Tests (`playwright/`)
```
playwright/tests/
  01-home.spec.ts            - Login, home, navigation
  02-kanban-board.spec.ts    - Board display
  03-task-management.spec.ts - Task CRUD
  04-roles-features.spec.ts  - Roles & features
  05-backlog-settings.spec.ts - Backlog & settings
  06-skills-stats-export.spec.ts - Skills, stats, export
  07-theme-comments-ws-health.spec.ts - Theme, comments, WebSocket
  helpers.ts                 - Shared test helpers
```
