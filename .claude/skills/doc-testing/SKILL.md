---
name: doc-testing
description: "Agach testing strategy: test contract pattern, mock structure, contract tests, testcontainers with postgres:17, security tests, QA seed data"
user-invocable: true
disable-model-invocation: false
---

# Agach Testing Strategy

## Test Contract Pattern
Each repository interface has a companion test package:
```
/domain/repositories/<name>/<name>.go        - Interface definition
/domain/repositories/<name>/<name>test/
    contract.go - Mock<Name> + <Name>ContractTesting
```

## Mock Structure
```go
type Mock<Name> struct {
    <Method>Func func(...) ...
}

func (m *Mock<Name>) <Method>(...) ... {
    if m.<Method>Func == nil {
        panic("called not defined <Method>Func")
    }
    return m.<Method>Func(...)
}
```

## Contract Test Structure
```go
func <Name>ContractTesting(t *testing.T, repo <Name>) {
    ctx := context.Background()

    t.Run("Contract: <behavior>", func(t *testing.T) {
        // Test implementation
        require.NoError(t, err)
        assert.ErrorIs(t, err, domain.ErrExpected)
    })
}
```

## Integration Tests
- Use `testcontainers-go` with `postgres:17` for all PostgreSQL tests
- One shared container per test package — start in `TestMain`, never per test function
- Pattern: `newTestPool()` → `setupRepos()` → contract testing

## Security Tests
- Located alongside handler tests (`*_security_test.go`, `deep_security_test.go`)
- Test auth enforcement, input validation, injection prevention
- Both command and query handlers have security test coverage

## Service Mock + Contract
```
domain/service/servicetest/contract.go - MockCommands + MockQueries
```
Used by app layer unit tests and inbound handler tests.

## QA Seed Data (`internal/server/qaseed/seed.go`)
Deterministic seed for E2E/Playwright tests. Wipes and recreates:
- 3 roles: Backend (⚙️), Frontend (🖥️), QA (🧪) with tech stacks
- 2 skills: Go Development, Playwright Testing
- 1 Dockerfile: "go-service" (Go 1.24 template)
- 2 projects: "QA Test Project" (main) + "QA Feature Branch" (feature)
- 8 tasks in various states: todo, in_progress, blocked, done, won't-do, backlog, feature-linked, dependency pair
- 2 comments: one on todo task, one human comment on blocked task
- Outputs JSON with all entity IDs for Playwright test fixtures

## Bug Fix Guidelines
- Add unit tests covering the fixed scenario to prevent regressions
- If the bug involves data/state that can be reproduced via seed data, add the use case to `internal/server/qaseed/`
- If the bug is reproducible through the web UI, add a Playwright test in `playwright/`

## Playwright Tests (`playwright/`)
```
playwright/tests/
  01-home.spec.ts            - Login, home, navigation
  02-kanban-board.spec.ts    - Board display
  03-task-management.spec.ts - Task CRUD
  04-roles-features.spec.ts  - Roles & features
  05-backlog-settings.spec.ts - Backlog & settings
  06-skills-stats-export.spec.ts - Skills, stats, export
  07-theme-comments-ws-health.spec.ts - Theme, comments, WebSocket
  helpers.ts                 - Shared test helpers
```

## data-qa Attributes
Interactive elements use `data-qa` attributes for Playwright targeting:
`create-project-btn`, `login-email-input`, `search-input`, `nav-kanban-btn`,
`task-open-btn`, `user-menu-btn`, `theme-toggle-btn`, etc.
