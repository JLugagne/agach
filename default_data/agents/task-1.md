# task-1 — Migration + domain types

## Context files
- `features/example-slug/FEATURE.md`
- `features/example-slug/scaffold-1_SUMMARY.md`
- `internal/kanban/domain/types.go`
- `internal/kanban/domain/errors.go`
- `internal/kanban/outbound/pg/migrations/001_schema.sql`

## Pre-check

`go build ./...` must pass before you write anything. The scaffold created stubs — if the build is broken, fix compilation errors first. Do not write feature code until the build is green.

## Goal

Add the domain types and errors for this feature. The scaffold created placeholder structs — replace them with the real field definitions. Add the SQL migration.

## What to do

### domain/types.go

The scaffold added a `FeatureID` type and `Feature` struct with empty bodies. Replace the empty bodies with the real definitions from FEATURE.md.

Follow the exact pattern of the nearest existing ID type and struct. JSON tags use snake_case. Time fields use `time.Time`. ID fields use the typed ID type, not `string`.

### domain/errors.go

The scaffold added empty error variables. Fill them in following the exact sentinel pattern used for existing errors in this file. Error codes are UPPER_SNAKE_CASE matching the variable name. Messages are lowercase readable strings.

### migrations/00N_feature.sql

Create a new migration file. Use `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` so the migration is idempotent on restart.

Follow the table structure implied by the domain type. UUIDs as PRIMARY KEY with `DEFAULT gen_random_uuid()`. All NOT NULL columns get sensible defaults. Enable RLS with a permissive policy using the `DO $$ BEGIN IF NOT EXISTS ... END $$` guard from migration 001.

Embed the migration in `pg.go` alongside the existing migrations and run it in `NewRepositories`.

## Done when

`go build ./internal/kanban/domain/...` passes.
`go build ./internal/kanban/outbound/pg/...` passes (migration embedded and runs).

## Output files

Create `features/example-slug/task-1_SUMMARY.md`:
- Files modified
- What was added to each (one line per addition)

Update `features/example-slug/TASKS.md`: mark task-1 `[x]`.
