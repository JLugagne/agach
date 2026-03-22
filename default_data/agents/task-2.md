# task-2 — FeatureRepository contract tests (RED)

## Context files
- `features/example-slug/task-1_SUMMARY.md`
- `internal/kanban/domain/repositories/features/features.go`
- `internal/kanban/domain/repositories/tasks/taskstest/contract.go` (pattern reference)

## Pre-check

`go build ./...` must pass before you write anything.

## Goal

Write contract tests for `FeatureRepository`. The scaffold created the interface and mock stubs. This task fills in the test cases and any missing mock method bodies.

## What to do

### featurestest/contract.go

The scaffold created `MockFeature` with nil function fields and `FeatureContractTesting` as an empty function. Fill them in:

**MockFeature** — each method delegates to its `*Func` field and panics with `"MockFeature.<MethodName> not set"` if nil. Follow the exact pattern from `taskstest/contract.go`.

**FeatureContractTesting** — write sub-tests using `t.Run`. Tests that require a real repository (Create, FindByID, List, Update, Delete) must use `t.Skip("requires real repository — run in pg integration test")`. Do not try to call the mock directly in contract testing.

Only write tests for mock behavior that can be verified without a real DB:
- `"mock FindBySlug returns nil for unknown slug"` — set FindBySlugFunc to return (nil, nil), call it, assert result is nil and err is nil
- `"mock Delete returns ErrFeatureInUse"` — set DeleteFunc to return ErrFeatureInUse, call it, assert ErrorIs

Everything else skips. The real contract tests run in task-4 against Postgres.

This is RED state by design — the skips are correct, not a problem to fix.

## Done when

`go build ./internal/kanban/domain/repositories/features/...` passes.
`go test ./internal/kanban/domain/repositories/features/...` passes (all sub-tests either pass or skip — no failures, no panics).

## Output files

Create `features/example-slug/task-2_SUMMARY.md`. Update TASKS.md: mark task-2 `[x]`.
