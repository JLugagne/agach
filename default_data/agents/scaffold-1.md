# scaffold-1 — Create all file skeletons

## Context files
- `features/example-slug/FEATURE.md`
- `internal/kanban/domain/types.go`
- `internal/kanban/domain/errors.go`
- `internal/kanban/domain/repositories/roles/roles.go` (pattern reference)
- `internal/kanban/outbound/pg/pg.go` (pattern reference)
- `internal/kanban/app/app.go` (pattern reference)
- `internal/kanban/inbound/commands/roles.go` (pattern reference)
- `internal/kanban/inbound/queries/roles.go` (pattern reference)

## Goal

Create every new file listed in FEATURE.md → "New files" section as an empty skeleton.
No logic. No tests. The build must pass when done.

## Rules

- Read each pattern reference file to understand conventions before writing anything
- Every new file gets correct package declaration and imports
- Every new interface has all methods listed with correct signatures — all return zero values
- Every new struct is defined with all fields
- Every new function on an existing type returns zero values only: `return nil`, `return domain.Feature{}, nil`, `return 0, nil`, `""`, `false`
- Every interface implementation gets a compile-time check: `var _ InterfaceName = (*StructName)(nil)`
- Existing files get only new fields/methods appended — do not touch existing code
- Do not write test files — tests are written in RED tasks
- Do not write SQL migration files — migrations are written in GREEN tasks

## What to create

For each file in FEATURE.md → "New files (hint for scaffolding)":

### Backend new files

Read `internal/kanban/domain/repositories/roles/roles.go` to understand interface conventions.
Read `internal/kanban/outbound/pg/pg.go` to understand repository struct conventions.
Read `internal/kanban/app/roles.go` to understand app method conventions.
Read `internal/kanban/inbound/commands/roles.go` to understand handler conventions.

Create each listed file with:
- Package declaration matching directory
- Required imports (infer from what the stubs reference)
- Empty structs, interfaces, and stub functions

### Existing files to modify

Append only — do not touch existing code:
- Add new fields to structs
- Add new methods to interfaces
- Add new method stubs to types that implement those interfaces

## Done when

`go build ./...` passes with no errors from repo root.

The build passing is the ONLY acceptance criterion. If it fails, fix the compilation error before completing.

## Output files

Create `features/example-slug/scaffold-1_SUMMARY.md`:
- List every file created or modified
- For each file: what was added (one line per addition)

Update `features/example-slug/TASKS.md`: mark scaffold-1 `[x]`.
