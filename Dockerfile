# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/ux
COPY ux/package.json ux/package-lock.json ./
RUN npm ci
COPY ux/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY internal/ internal/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY ux/embed.go ux/embed.go
COPY --from=frontend /app/ux/dist ux/dist/
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
