---
name: doc-notifications
description: "Agach notification system: scopes (project/agent/global), severity levels, delivery via WebSocket and REST API, notification triggers"
user-invocable: true
disable-model-invocation: false
---

# Agach Notification System

## Overview

Notifications are user-facing alerts triggered by system events (feature done, task blocked, etc.). Each notification has a **scope** (project, agent, or global), a severity, content, and an optional action link. Delivered via WebSocket hub and REST API.

## Scopes

| Scope     | `project_id` | `agent_slug` | Use case                                      |
|-----------|-------------|--------------|-----------------------------------------------|
| `project` | required    | optional     | Feature done, task blocked within a project   |
| `agent`   | optional    | required     | Agent-specific alerts (build failure, etc.)    |
| `global`  | nil         | empty        | System-wide announcements, maintenance, etc.  |

## Domain Types

```go
// internal/server/domain/types.go

type NotificationScope string    // "project", "agent", "global"
type NotificationSeverity string // "info", "success", "warning", "error"

type Notification struct {
    ID        NotificationID
    ProjectID *ProjectID          // nil for global
    Scope     NotificationScope
    AgentSlug string              // set for agent-scoped
    Severity  NotificationSeverity
    Title     string
    Text      string
    LinkURL   string
    LinkText  string
    LinkStyle string              // primary, secondary, danger, warning
    ReadAt    *time.Time
    CreatedAt time.Time
}
```

## Severity Levels

| Severity  | Use case                                  | Frontend color |
|-----------|-------------------------------------------|----------------|
| `info`    | Informational (feature status changed)    | Blue           |
| `success` | Positive outcome (feature done, task done)| Green          |
| `warning` | Needs attention (task blocked)            | Yellow/Orange  |
| `error`   | Critical issue (all tasks blocked)        | Red            |

## REST API

### Global endpoints
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
h.hub.Broadcast(websocket.Event{Type: "notification", ProjectID: string(*notification.ProjectID), Data: resp})

// Global/agent-scoped: all connected clients receive it
h.hub.Broadcast(websocket.Event{Type: "notification", Data: resp})
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

## Architecture Placement

```
internal/server/
  domain/
    types.go                    — Notification, NotificationID, NotificationSeverity, NotificationScope
    errors.go                   — ErrNotificationNotFound, ErrInvalidNotificationData, ErrNotificationTitleRequired
    repositories/notifications/ — Repository interface + NotificationFilters
    repositories/notifications/notificationstest/ — Mock + contract tests
  app/
    notifications.go            — Create (validates title/severity/scope), list, mark-read, delete
  inbound/
    commands/notifications.go   — POST (project + global), PUT (mark read, mark all read), DELETE
    queries/notifications.go    — GET list (global + project), GET unread-count (global + project)
    converters/notifications.go — Domain ↔ API converters
  outbound/
    pg/pg_notifications.go      — PostgreSQL implementation with dynamic filter builder
    pg/migrations/003_notifications.sql
```
