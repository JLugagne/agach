---
name: doc-notifications
description: "Agach notification system: scopes (project/agent/global), severity levels, delivery via WebSocket and REST API, notification triggers"
user-invocable: true
disable-model-invocation: false
---

# Agach Notification System

## Overview

Notifications are user-facing alerts triggered by system events (feature done, task blocked, etc.). Each notification has a **scope** (project, agent, or global), a severity, content, and an optional action link. All notifications appear in a global feed; they can also be filtered by project, scope, or agent. Delivered via WebSocket hub and REST API.

## Scopes

| Scope     | `project_id` | `agent_slug` | Use case                                      |
|-----------|-------------|--------------|-----------------------------------------------|
| `project` | required    | optional     | Feature done, task blocked within a project   |
| `agent`   | optional    | required     | Agent-specific alerts (build failure, etc.)    |
| `global`  | nil         | empty        | System-wide announcements, maintenance, etc.  |

## Domain Types

```go
// internal/kanban/domain/types.go

type NotificationScope string

const (
    NotificationScopeProject NotificationScope = "project"
    NotificationScopeAgent   NotificationScope = "agent"
    NotificationScopeGlobal  NotificationScope = "global"
)

type NotificationSeverity string

const (
    SeverityInfo    NotificationSeverity = "info"
    SeveritySuccess NotificationSeverity = "success"
    SeverityWarning NotificationSeverity = "warning"
    SeverityError   NotificationSeverity = "error"
)

type Notification struct {
    ID        NotificationID       `json:"id"`
    ProjectID *ProjectID           `json:"project_id,omitempty"` // nil for global notifications
    Scope     NotificationScope    `json:"scope"`
    AgentSlug string               `json:"agent_slug,omitempty"` // set for agent-scoped
    Severity  NotificationSeverity `json:"severity"`
    Title     string               `json:"title"`
    Text      string               `json:"text"`
    LinkURL   string               `json:"link_url,omitempty"`
    LinkText  string               `json:"link_text,omitempty"`
    LinkStyle string               `json:"link_style,omitempty"` // primary, secondary, danger, warning
    ReadAt    *time.Time           `json:"read_at"`
    CreatedAt time.Time            `json:"created_at"`
}
```

## Severity Levels

| Severity  | Use case                                  | Frontend color |
|-----------|-------------------------------------------|----------------|
| `info`    | Informational (feature status changed)    | Blue           |
| `success` | Positive outcome (feature done, task done)| Green          |
| `warning` | Needs attention (task blocked)            | Yellow/Orange  |
| `error`   | Critical issue (all tasks blocked)        | Red            |

## Repository Interface

```go
// internal/kanban/domain/repositories/notifications/notifications.go

type NotificationFilters struct {
    ProjectID  *domain.ProjectID
    Scope      *domain.NotificationScope
    AgentSlug  string
    UnreadOnly bool
}

type NotificationRepository interface {
    Create(ctx context.Context, notification domain.Notification) error
    FindByID(ctx context.Context, id domain.NotificationID) (*domain.Notification, error)
    List(ctx context.Context, filters NotificationFilters, limit, offset int) ([]domain.Notification, error)
    UnreadCount(ctx context.Context, filters NotificationFilters) (int, error)
    MarkRead(ctx context.Context, id domain.NotificationID) error
    MarkAllRead(ctx context.Context, filters NotificationFilters) error
    Delete(ctx context.Context, id domain.NotificationID) error
}
```

## REST API

### Global endpoints (all notifications)

```
POST   /api/notifications                              — Create (scope from body: agent or global)
GET    /api/notifications?scope=&agent_slug=&unread=&limit=&offset= — List all
GET    /api/notifications/unread-count?scope=&agent_slug= — Global unread count
PUT    /api/notifications/{id}/read                    — Mark single as read
PUT    /api/notifications/read-all                     — Mark all as read
DELETE /api/notifications/{id}                         — Delete
```

### Project-scoped endpoints

```
POST   /api/projects/{id}/notifications                — Create (defaults scope=project)
GET    /api/projects/{id}/notifications?scope=&agent_slug=&unread=&limit=&offset= — List for project
GET    /api/projects/{id}/notifications/unread-count    — Unread count for project
PUT    /api/projects/{id}/notifications/read-all        — Mark all project notifications as read
```

## WebSocket Delivery

```go
// Project-scoped: only clients subscribed to that project receive it
h.hub.Broadcast(websocket.Event{
    Type:      "notification",
    ProjectID: string(*notification.ProjectID),
    Data:      resp,
})

// Global/agent-scoped: all connected clients receive it
h.hub.Broadcast(websocket.Event{
    Type: "notification",
    Data: resp,
})
```

## Database Schema (003_notifications.sql)

```sql
CREATE TABLE notifications (
    id          TEXT PRIMARY KEY CHECK (is_valid_uuid(id)),
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,  -- nullable
    scope       TEXT NOT NULL DEFAULT 'project' CHECK (scope IN ('project', 'agent', 'global')),
    agent_slug  TEXT NOT NULL DEFAULT '',
    severity    TEXT NOT NULL CHECK (severity IN ('info', 'success', 'warning', 'error')),
    title       TEXT NOT NULL,
    text        TEXT NOT NULL,
    link_url    TEXT NOT NULL DEFAULT '',
    link_text   TEXT NOT NULL DEFAULT '',
    link_style  TEXT NOT NULL DEFAULT '',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Notification Triggers

| Trigger                        | Scope     | Severity  | Title example                        |
|--------------------------------|-----------|-----------|--------------------------------------|
| Feature status → `done`        | project   | `success` | "Feature completed"                  |
| Feature status → `blocked`     | project   | `warning` | "Feature blocked"                    |
| Task blocked                   | project   | `warning` | "Task blocked: {task title}"         |
| Task won't-do requested        | project   | `warning` | "Won't-do requested: {task title}"   |
| All feature tasks completed    | project   | `success` | "All tasks done for: {feature name}" |
| Agent build failure            | agent     | `error`   | "Build failed for agent X"           |
| System maintenance             | global    | `info`    | "Scheduled maintenance"              |

## Frontend TypeScript Type

```ts
interface Notification {
    id: string;
    project_id?: string;
    scope: 'project' | 'agent' | 'global';
    agent_slug?: string;
    severity: 'info' | 'success' | 'warning' | 'error';
    title: string;
    text: string;
    link_url?: string;
    link_text?: string;
    link_style?: 'primary' | 'secondary' | 'danger' | 'warning';
    read_at: string | null;
    created_at: string;
}
```

## Architecture Placement

```
internal/kanban/
  domain/
    types.go                    — Notification, NotificationID, NotificationSeverity, NotificationScope
    errors.go                   — ErrNotificationNotFound, ErrInvalidNotificationData, ErrNotificationTitleRequired
    repositories/notifications/ — Repository interface + NotificationFilters
    repositories/notifications/notificationstest/ — Mock + contract tests (20 tests)
  app/
    notifications.go            — Create (validates title/severity/scope), list, mark-read, delete
    notifications_test.go       — 8 unit tests
  inbound/
    commands/notifications.go   — POST (project + global), PUT (mark read, mark all read), DELETE
    queries/notifications.go    — GET list (global + project), GET unread-count (global + project)
    converters/notifications.go — Domain ↔ API converters
  outbound/
    pg/pg_notifications.go      — PostgreSQL implementation with dynamic filter builder
    pg/migrations/003_notifications.sql
```
