# Tasks

## Legend
```
[ ] todo
[-] in progress
[x] done
[!] error / blocked
```

## Status

```
[ ] scaffold-1
[ ] task-1
[ ] task-2
[ ] task-3
[ ] task-4
[ ] task-5
[ ] task-6
```

---

## Details

### scaffold-1
agent: scaffolding
status: [ ]
children:
  - task-1

summary:
  Create all new files with empty structs, interfaces, and function stubs.
  go build ./... must pass when done. No logic. No tests.

---

### task-1
agent: go-implement
status: [ ]
parents:
  - scaffold-1
children:
  - task-2
  - task-3

summary:
  Migration + domain types. Add FeatureID, Feature struct, domain errors.
  go build ./internal/kanban/domain/... must pass.

---

### task-2
agent: go-postgres
status: [ ]
parents:
  - task-1
children:
  - task-4

summary:
  Contract tests for FeatureRepository (RED phase).
  Tests skip with t.Skip until pg implementation exists.

---

### task-3
agent: go-implement
status: [ ]
parents:
  - task-1
children:
  - task-4

summary:
  Service interface extensions + mock contract tests.
  go build ./internal/kanban/domain/service/... must pass.

---

### task-4
agent: go-postgres
status: [ ]
parents:
  - task-2
  - task-3
children:
  - task-5

summary:
  Postgres implementation (GREEN). All contract tests pass.

---

### task-5
agent: go-implement
status: [ ]
parents:
  - task-4
children:
  - task-6

summary:
  App layer + HTTP handlers + converters. go build ./... must pass.

---

### task-6
agent: go-implement
status: [ ]
parents:
  - task-5
children: []

summary:
  Frontend: FeaturesPage + types + API functions. npm run build must pass.
