---
name: doc-dependencies
description: "Agach key dependencies: Go libraries (uuid, gorilla/mux, pgx/v5, jwt, testify, testcontainers, MCP SDK, validator, logrus, crypto, docker, go-git, sqlite)"
user-invocable: true
disable-model-invocation: false
---

# Agach Key Dependencies

## Direct Dependencies
```
- github.com/google/uuid               - UUID generation (UUIDv7)
- github.com/gorilla/mux               - HTTP routing
- github.com/gorilla/websocket         - WebSocket
- github.com/jackc/pgx/v5              - PostgreSQL driver + pgxpool
- github.com/golang-jwt/jwt/v5         - JWT token generation/validation
- github.com/go-playground/validator/v10 - Request validation (struct tags)
- github.com/sirupsen/logrus           - Structured logging
- github.com/stretchr/testify          - Testing (assert + require)
- github.com/modelcontextprotocol/go-sdk - MCP server
- github.com/go-git/go-git/v5          - Git operations (clone, fetch, pull, worktrees)
- github.com/docker/docker             - Docker daemon integration (28.5.2)
- modernc.org/sqlite                   - SQLite database (daemon local storage)
- golang.org/x/crypto                  - bcrypt password hashing
- golang.org/x/time                    - Rate limiting
- gopkg.in/yaml.v3                     - YAML config parsing
```

## Test Dependencies
```
- github.com/testcontainers/testcontainers-go              - Container management
- github.com/testcontainers/testcontainers-go/modules/postgres - PostgreSQL test container (postgres:17)
```

## NOT Used
```
- github.com/mattn/go-sqlite3     - Removed (replaced by modernc.org/sqlite, CGO-free)
```
