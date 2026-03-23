---
name: doc-dependencies
description: "Agach key dependencies: Go libraries (uuid, gorilla/mux, pgx/v5, jwt, testify, testcontainers, MCP SDK, validator, logrus, crypto)"
user-invocable: true
disable-model-invocation: false
---

# Agach Key Dependencies

## Direct Dependencies
```
- github.com/google/uuid          - UUID generation (UUIDv7)
- github.com/gorilla/mux          - HTTP routing
- github.com/gorilla/websocket    - WebSocket
- github.com/jackc/pgx/v5         - PostgreSQL driver + pgxpool
- github.com/golang-jwt/jwt/v5    - JWT token generation/validation
- github.com/go-playground/validator/v10 - Request validation
- github.com/sirupsen/logrus      - Logging
- github.com/stretchr/testify     - Testing (assert + require)
- github.com/modelcontextprotocol/go-sdk - MCP server
- github.com/gdamore/tcell/v2     - TUI rendering
- golang.org/x/crypto             - bcrypt password hashing
- golang.org/x/time               - Rate limiting
- gopkg.in/yaml.v3                - YAML config parsing
```

## Test Dependencies
```
- github.com/testcontainers/testcontainers-go              - Container management
- github.com/testcontainers/testcontainers-go/modules/postgres - PostgreSQL test container (postgres:17)
```

## NOT Used
```
- github.com/mattn/go-sqlite3     - Removed (migrated to PostgreSQL)
```
