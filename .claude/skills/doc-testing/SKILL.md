---
name: doc-testing
description: "Agach testing strategy: test contract pattern, mock structure, contract tests, testcontainers with postgres:17, bug fix guidelines"
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

## Bug Fix Guidelines
- Add unit tests covering the fixed scenario to prevent regressions
- If the bug involves data/state that can be reproduced via seed data, add the use case to `cmd/qa-seed/`
- If the bug is reproducible through the web UI, add a Playwright test in `playwright/`
