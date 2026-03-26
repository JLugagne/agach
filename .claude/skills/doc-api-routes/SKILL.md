---
name: doc-api-routes
description: "Agach complete HTTP API reference: all REST endpoints, request/response types, WebSocket events, SSE, query parameters"
user-invocable: true
disable-model-invocation: false
---

# Agach HTTP API Reference

All routes require JWT Bearer auth unless noted. Server runs on port 8322.

## Projects
```
POST   /api/projects                          — Create project
GET    /api/projects                          — List projects (with summary)
GET    /api/projects/{id}                     — Get project
GET    /api/projects/{id}/info                — Get project info (with breadcrumb, children)
GET    /api/projects/{id}/summary             — Get project task counts
GET    /api/projects/{id}/children            — List sub-projects
PATCH  /api/projects/{id}                     — Update project
DELETE /api/projects/{id}                     — Delete project (cascades)
```

## Tasks
```
POST   /api/projects/{id}/tasks               — Create task
GET    /api/projects/{id}/tasks               — List tasks (?column=&assigned_role=&priority=&tag=&search=&limit=&offset=)
GET    /api/projects/{id}/tasks/search        — Search tasks (?q=&limit=)
GET    /api/projects/{id}/tasks/{taskId}      — Get task
GET    /api/projects/{id}/board               — Get board (columns with tasks)
GET    /api/projects/{id}/columns             — List columns
GET    /api/projects/{id}/next-tasks          — Get next available tasks (?count=&role=&feature_id=)
PATCH  /api/projects/{id}/tasks/{taskId}      — Update task
DELETE /api/projects/{id}/tasks/{taskId}      — Delete task
POST   /api/projects/{id}/tasks/{taskId}/move — Move task (body: target_column, reason)
POST   /api/projects/{id}/tasks/{taskId}/move-to-project — Move to another project
POST   /api/projects/{id}/tasks/{taskId}/reorder — Reorder task (body: position)
POST   /api/projects/{id}/tasks/{taskId}/complete — Complete task (body: completion_summary, files_modified, tokens)
POST   /api/projects/{id}/tasks/{taskId}/unblock — Unblock task
POST   /api/projects/{id}/tasks/{taskId}/wont-do — Request won't-do
POST   /api/projects/{id}/tasks/{taskId}/approve-wont-do — Approve won't-do
POST   /api/projects/{id}/tasks/{taskId}/reject-wont-do — Reject won't-do
PATCH  /api/projects/{id}/tasks/{taskId}/session — Update session ID
POST   /api/projects/{id}/tasks/{taskId}/seen — Mark task as seen
```

## Comments
```
POST   /api/projects/{id}/tasks/{taskId}/comments              — Create comment
GET    /api/projects/{id}/tasks/{taskId}/comments              — List comments (?limit=&offset=)
PATCH  /api/projects/{id}/tasks/{taskId}/comments/{commentId}  — Update comment
DELETE /api/projects/{id}/tasks/{taskId}/comments/{commentId}  — Delete comment
```

## Dependencies
```
GET    /api/projects/{id}/tasks/{taskId}/dependencies — List dependencies
GET    /api/projects/{id}/tasks/{taskId}/dependents   — List dependents
```

## Agents (Roles)
```
POST   /api/agents                            — Create global agent
GET    /api/agents                            — List global agents
GET    /api/agents/{slug}                     — Get agent by slug
PATCH  /api/agents/{slug}                     — Update agent
DELETE /api/agents/{slug}                     — Delete agent
POST   /api/agents/{slug}/clone               — Clone agent (body: new_slug, new_name)
```

## Specialized Agents
```
POST   /api/agents/{slug}/specialized                    — Create specialized agent
GET    /api/agents/{slug}/specialized                    — List specialized agents
GET    /api/agents/{slug}/specialized/{specSlug}         — Get specialized agent
PATCH  /api/agents/{slug}/specialized/{specSlug}         — Update specialized agent
DELETE /api/agents/{slug}/specialized/{specSlug}         — Delete specialized agent
GET    /api/agents/{slug}/specialized/{specSlug}/skills  — List specialized agent skills
```

## Project Agents
```
POST   /api/projects/{projectId}/agents                  — Assign agent to project
GET    /api/projects/{projectId}/agents                  — List project agents
GET    /api/projects/{projectId}/agents/{slug}/tasks     — List tasks by agent
DELETE /api/projects/{projectId}/agents/{slug}           — Remove agent from project
POST   /api/projects/{projectId}/agents/bulk-reassign    — Bulk reassign tasks
```

## Skills
```
POST   /api/skills                            — Create skill
GET    /api/skills                            — List skills
GET    /api/skills/{slug}                     — Get skill
PATCH  /api/skills/{slug}                     — Update skill
DELETE /api/skills/{slug}                     — Delete skill
POST   /api/agents/{slug}/skills              — Add skill to agent
GET    /api/agents/{slug}/skills              — List agent skills
DELETE /api/agents/{slug}/skills/{skillSlug}  — Remove skill from agent
```

## Features
```
POST   /api/projects/{id}/features                          — Create feature
GET    /api/projects/{id}/features                          — List features (?status=)
GET    /api/projects/{id}/features/{featureId}              — Get feature
PATCH  /api/projects/{id}/features/{featureId}              — Update feature
PATCH  /api/projects/{id}/features/{featureId}/status       — Update feature status
DELETE /api/projects/{id}/features/{featureId}              — Delete feature
GET    /api/projects/{id}/stats/features                    — Get feature stats
```

## Dockerfiles
```
POST   /api/dockerfiles                       — Create dockerfile
GET    /api/dockerfiles                       — List dockerfiles
GET    /api/dockerfiles/{id}                  — Get dockerfile
PATCH  /api/dockerfiles/{id}                  — Update dockerfile
DELETE /api/dockerfiles/{id}                  — Delete dockerfile
PUT    /api/projects/{id}/dockerfile          — Set project dockerfile
GET    /api/projects/{id}/dockerfile          — Get project dockerfile
DELETE /api/projects/{id}/dockerfile          — Clear project dockerfile
```

## Notifications
```
POST   /api/notifications                     — Create global/agent notification
GET    /api/notifications                     — List all (?scope=&agent_slug=&unread=&limit=&offset=)
GET    /api/notifications/unread-count        — Global unread count
PUT    /api/notifications/{id}/read           — Mark as read
PUT    /api/notifications/read-all            — Mark all as read
DELETE /api/notifications/{id}                — Delete notification
POST   /api/projects/{id}/notifications       — Create project notification
GET    /api/projects/{id}/notifications       — List project notifications
GET    /api/projects/{id}/notifications/unread-count — Project unread count
PUT    /api/projects/{id}/notifications/read-all     — Mark all project as read
```

## Chat Sessions
```
POST   /api/projects/{projectId}/features/{featureId}/chats                        — Start session
GET    /api/projects/{projectId}/features/{featureId}/chats                        — List sessions
GET    /api/projects/{projectId}/features/{featureId}/chats/{sessionId}            — Get session
POST   /api/projects/{projectId}/features/{featureId}/chats/{sessionId}/end        — End session
POST   /api/projects/{projectId}/features/{featureId}/chats/{sessionId}/upload     — Upload JSONL
GET    /api/projects/{projectId}/features/{featureId}/chats/{sessionId}/download   — Download JSONL
```

## Statistics
```
GET    /api/projects/{id}/tool-usage          — Tool usage stats
GET    /api/projects/{id}/stats/timeline      — Timeline (?days=60)
GET    /api/projects/{id}/stats/cold-start    — Cold start token stats
GET    /api/projects/{id}/stats/model-tokens  — Model token stats
GET    /api/model-pricing                     — Model pricing list
```

## Images (currently unavailable)
```
POST   /api/projects/{id}/images              — Upload image (returns error)
GET    /api/projects/{id}/images/{filename}   — Serve image (returns error)
```

## Real-time
```
GET    /api/projects/{id}/sse                 — Server-Sent Events stream
WS     /ws                                    — WebSocket (registered without auth middleware)
```

## Auth (see /doc-identity for full details)
```
POST   /api/auth/register, /api/auth/login, /api/auth/refresh, /api/auth/logout
GET    /api/auth/me, PATCH /api/auth/me, POST /api/auth/me/password
GET    /api/auth/sso/providers, /api/auth/sso/{provider}/authorize, /api/auth/sso/{provider}/callback
```

## WebSocket Event Types
Task: `task_created`, `task_updated`, `task_deleted`, `task_moved`, `task_completed`
Project: `project_created`, `project_updated`, `project_deleted`
Comment: `comment_added`, `comment_edited`
Agent: `agent_created`, `agent_updated`, `agent_deleted`
Skill: `skill_created`, `skill_updated`, `skill_deleted`
Feature: `feature_created`, `feature_updated`, `feature_status_updated`, `feature_deleted`
Other: `notification`, `wip_slot_available`, `wont_do_rejected`, `task_wont_do`
