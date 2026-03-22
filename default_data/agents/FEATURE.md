# Feature: [Title]

**Slug**: `example-slug`
**Status**: planned

## Problem

One paragraph. What is broken or missing and why it matters.
No solution language here — just the problem.

## Acceptance criteria

Each criterion must be independently testable. An agent must be able to write a test for it.

- [ ] When X happens, Y is the result
- [ ] Z is persisted to the database with these fields
- [ ] The API returns this response shape on this endpoint

## Out of scope

Explicit list. Forces the planner to acknowledge boundaries before creating tasks.

- No authentication or RBAC
- No versioning
- No pagination on the first pass

## Constraints

Technical or business rules the implementation must respect.

- Must not change existing public API signatures
- Must be backward compatible with existing data
- Response time must stay under 100ms on the hot path

## Open questions

Must be empty before planning begins. Each question must be specific enough that a one-sentence answer resolves it.

- (none)

## Key design decisions

| Decision | Choice | Rationale |
|---|---|---|
| Storage | Postgres row | Enables queries and joins with existing tables |
| ID format | UUIDv7 | Time-ordered, native Postgres type |

## New files (hint for scaffolding)

List every new file the feature will create. The scaffolding agent uses this to create
empty skeletons before any RED/GREEN work begins. Be complete — missing a file here
means an agent creates it from scratch with full exploration cost.

### Backend
- `internal/kanban/domain/types.go` — add FeatureID type and Feature struct
- `internal/kanban/domain/errors.go` — add ErrFeatureNotFound, ErrFeatureAlreadyExists
- `internal/kanban/domain/repositories/features/features.go` — FeatureRepository interface
- `internal/kanban/domain/repositories/features/featurestest/contract.go` — MockFeature + contract tests
- `internal/kanban/outbound/pg/migrations/003_features.sql` — schema migration
- `internal/kanban/outbound/pg/pg_features.go` — pg implementation
- `internal/kanban/app/features.go` — app layer methods
- `internal/kanban/inbound/converters/features.go` — ToPublicFeature converter
- `internal/kanban/inbound/commands/features.go` — HTTP command handlers
- `internal/kanban/inbound/queries/features.go` — HTTP query handlers
- `pkg/kanban/types.go` — add CreateFeatureRequest, FeatureResponse

### Frontend
- `ux/src/pages/FeaturesPage.tsx` — full CRUD page
- `ux/src/components/FeatureModal.tsx` — create/edit modal

### Existing files modified
- `internal/kanban/domain/service/commands.go` — add feature commands to interface
- `internal/kanban/domain/service/queries.go` — add feature queries to interface
- `internal/kanban/app/app.go` — add features field to App struct
- `internal/kanban/init.go` — register feature handlers
- `ux/src/lib/types.ts` — add FeatureResponse type
- `ux/src/lib/api.ts` — add feature API functions
