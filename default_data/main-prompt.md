please implement the feature @features/{{feature-slug}}/FEATURE.md

## How to read the task files

Each task has its own file at `features/{{feature-slug}}/task-N.md` (and `scaffold-1.md`).
The task file is the complete specification — read it fully before starting that task.

## Execution rules

### 1. Always start with the pre-check

Before writing any code for any task, run:
- Backend tasks: `go build ./...` from repo root
- Frontend tasks: `npm run build` inside `ux/`

If the build fails for a reason unrelated to your task, block immediately by writing `features/{{feature-slug}}/BLOCKED.md` with the exact error output. Do not proceed.

### 2. Run tasks in dependency order

Read `features/{{feature-slug}}/TASKS.md` to understand the dependency graph.
Always run the scaffold task first. A task may not start until all its parents are marked `[x]`.

Run tasks in parallel only when they have no dependency relationship — check the `children` / `parents` fields in TASKS.md.

### 3. scaffold-1 runs before everything

The scaffolding task creates all new files as empty skeletons so that `go build ./...` passes from the start. It uses only mechanical work — no logic, no tests, no SQL. It must complete and the build must pass before any other task begins.

### 4. One summary file per task

When a task is done, create `features/{{feature-slug}}/task-N_SUMMARY.md` listing:
- Every file created or modified
- What was added (one line per item, no code blocks)

Update `features/{{feature-slug}}/TASKS.md` to mark the task `[x]`.

### 5. Circuit breaker — stop on loops

If you find yourself running the same command more than 3 times in a row with the same result, stop. Write `features/{{feature-slug}}/BLOCKED.md` with:
- The command you are running
- The output it produces
- What you expected instead

Do not attempt more fixes. The block is the correct output.

### 6. Tests must pass before marking done

- Backend tasks: the test command in the task file must pass — do not mark done with failing tests
- RED tasks: `go test` must pass (all skips are fine, failures are not)
- GREEN tasks: all previously skipped tests must now pass

### 7. Never touch unrelated code

Each task has a defined scope. If you notice a bug or improvement outside that scope, note it in the summary file under "Observed but out of scope." Do not fix it.

## Start

Read `features/{{feature-slug}}/TASKS.md` first. Then read `features/{{feature-slug}}/scaffold-1.md` and begin.
