# Resource Distribution

The sidecar binary is embedded into the server at build time and distributed to daemons at runtime via a SHA-512 checksum-based sync protocol.

## Build-Time Flow

```mermaid
sequenceDiagram
    participant Docker as Dockerfile
    participant Sidecar as agach-sidecar
    participant Resources as resources/
    participant Server as agach-server

    Docker->>Sidecar: CGO_ENABLED=0 go build -o resources/agach-sidecar
    Sidecar-->>Resources: Binary written to resources/
    Docker->>Server: go build ./cmd/agach-server
    Note over Server: resources/embed.go: //go:embed *
    Resources-->>Server: agach-sidecar embedded via go:embed
```

## Runtime Flow

```mermaid
sequenceDiagram
    participant Server as agach-server
    participant WS as WebSocket
    participant Daemon as agach-daemon
    participant Cache as ~/.cache/agach/resources/
    participant API as GET /api/resources/{name}

    Note over Server: Startup: compute SHA-512<br/>for each embedded resource

    Daemon->>WS: Connect (JWT auth)
    Server->>WS: resource_manifest event
    WS->>Daemon: [{name, sha512, size}, ...]

    Daemon->>Daemon: Compare manifest SHA-512<br/>with local cache

    alt SHA-512 matches
        Note over Daemon: Resource up to date, skip
    else SHA-512 differs or missing
        Daemon->>API: GET /api/resources/agach-sidecar
        API-->>Daemon: Binary data
        Daemon->>Daemon: Verify SHA-512 of download
        Daemon->>Cache: Write to ~/.cache/agach/resources/agach-sidecar
        Note over Cache: chmod 0755 (executable)
    end
```

## Chat Session Flow

```mermaid
sequenceDiagram
    participant Daemon as agach-daemon
    participant Cache as ResourceCache
    participant Proxy as SidecarProxy (Unix socket)
    participant Claude as claude CLI
    participant Sidecar as agach-sidecar

    Daemon->>Cache: GetPath("agach-sidecar")
    Cache-->>Daemon: ~/.cache/agach/resources/agach-sidecar

    Daemon->>Proxy: Start (Unix socket)
    Daemon->>Claude: Spawn with env vars:<br/>AGACH_PROXY, AGACH_PROXY_KEY, AGACH_FEATURE_ID

    Note over Claude: MCP config references<br/>agach-sidecar binary path

    Claude->>Sidecar: Spawn via MCP stdio
    Sidecar->>Proxy: HTTP over Unix socket
    Proxy->>Proxy: Inject Bearer token,<br/>project ID, feature ID
    Proxy-->>Server: Forward to agach-server
```

## Security

- Resources are served over authenticated endpoints (JWT required)
- SHA-512 verification after download prevents tampering in transit
- The daemon only accepts resources whose hash matches the server-provided manifest
- The manifest is sent over the authenticated WebSocket connection
- Socket files are `chmod 0600` (owner-only access)
- Each sidecar session gets a unique cryptographic API key
