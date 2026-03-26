# Stage 1: Build frontend
FROM node:25-alpine AS frontend
WORKDIR /app/internal/server/ux
COPY internal/server/ux/package.json internal/server/ux/package-lock.json ./
RUN npm ci
COPY internal/server/ux/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY internal/ internal/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY resources/ resources/
COPY --from=frontend /app/internal/server/ux/dist internal/server/ux/dist/
# Build sidecar into resources/ so it gets embedded into the server binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o resources/agach-sidecar ./cmd/agach-sidecar
RUN go build -ldflags="-extldflags '-static'" -o /agach-server ./cmd/agach-server

# Stage 3: Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates sqlite
WORKDIR /app
COPY --from=backend /agach-server .
ENV AGACH_HOST=0.0.0.0
ENV AGACH_PORT=8322
ENV AGACH_MCP_HOST=0.0.0.0
ENV AGACH_MCP_PORT=8323
# Data stored in os.UserConfigDir()/agach-mcp/ by default
# Override with AGACH_DATA_DIR if needed
ENV AGACH_DATA_DIR=/data
EXPOSE 8322 8323
VOLUME ["/data"]
ENTRYPOINT ["/app/agach-server"]
