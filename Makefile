.PHONY: generate build build-server build-daemon build-sidecar run dev test clean docker docker_build run-playwright

GO_TAGS := sqlite_fts5

generate:
	cd internal/server/ux && npm run build

build: build-server

build-daemon:
	CGO_ENABLED=1 go build -tags $(GO_TAGS) -o agach-daemon ./cmd/agach-daemon

build-sidecar:
	CGO_ENABLED=0 go build -o resources/agach-sidecar ./cmd/agach-sidecar

build-server: generate build-sidecar
	go build -tags $(GO_TAGS) -o agach-server ./cmd/agach-server

run: build-server
	./agach-server

dev: generate
	CGO_ENABLED=1 go run -tags $(GO_TAGS) ./cmd/agach-server

test:
	CGO_ENABLED=1 go test -tags $(GO_TAGS) -race -failfast ./...

clean:
	rm -f agach-server agach-daemon
	rm -f resources/agach-sidecar
	rm -rf internal/server/ux/dist

docker:
	docker build -t agach-mcp .

docker_build:
	docker build -f Dockerfile.local --output type=local,dest=. .

run-playwright:
	docker compose -f docker-compose.playwright.yml up --build --abort-on-container-exit --exit-code-from playwright; \
	docker compose -f docker-compose.playwright.yml down -v
