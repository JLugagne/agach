# task-4 — Postgres implementation (GREEN)

## Context files
- `features/example-slug/task-1_SUMMARY.md`
- `features/example-slug/task-2_SUMMARY.md`
- `features/example-slug/task-3_SUMMARY.md`
- `internal/kanban/domain/repositories/features/features.go`
- `internal/kanban/domain/repositories/features/featurestest/contract.go`
- `internal/kanban/outbound/pg/pg.go`
- `internal/kanban/outbound/pg/pg_test.go` (read for testcontainers setup pattern)

## Pre-check

`go build ./...` must pass before you write anything. The scaffold created `pg_features.go` with stub methods — this task fills in the real SQL.

## Goal

Implement all `FeatureRepository` methods in `pg_features.go`. All skipped contract tests must pass against a real Postgres container.

## What to do

### pg_features.go

The scaffold created stub methods that return nil. Replace each stub with a real SQL implementation. Follow the exact query, scanning, and error-handling patterns from the nearest similar methods in `pg.go`.

Key patterns to match:
- Use `pgx.ErrNoRows` to detect not-found and return `(nil, nil)`
- Use `isUniqueViolation(err)` to detect constraint violations and return the appropriate domain error
- Use `isCheckViolation(err)` where relevant
- Scan JSONB columns the same way existing array columns are scanned
- Return `domain.ErrFeatureInUse` from Delete if the feature is referenced by other rows

### pg_features_test.go

Create a test file that calls the skipped contract tests from task-2 with a real repository:

```go
func TestFeatureContract(t *testing.T) {
    repo := setupTestRepositories(t)
    featurestest.FeatureContractTesting(t, repo.Features)
}
```

Also add the full contract tests that were skipped (Create, FindByID, List ordering, Update, Delete, Delete-ErrFeatureInUse) directly in this file using the same `setupTestRepositories` helper and real Postgres fixtures.

## Done when

`go test ./internal/kanban/outbound/pg/... -run TestFeatureContract -v`

Every sub-test must PASS. No skips. No failures. If a test fails, fix the implementation before marking done.

## Output files

Create `features/example-slug/task-4_SUMMARY.md`. Update TASKS.md: mark task-4 `[x]`.
