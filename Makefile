.PHONY: generate build build-server build-daemon run dev test clean docker docker_build

GO_TAGS := sqlite_fts5

generate:
	cd internal/server/ux && npm run build

build: build-server

build-daemon:
	CGO_ENABLED=1 go build -tags $(GO_TAGS) -o agach-daemon ./cmd/agach-daemon

build-server: generate
	go build -tags $(GO_TAGS) -o agach-server ./cmd/agach-server

run: build-server
	./agach-server

dev: generate
	CGO_ENABLED=1 go run -tags $(GO_TAGS) ./cmd/agach-server

test:
	CGO_ENABLED=1 go test -tags $(GO_TAGS) -race -failfast ./...

clean:
	rm -f agach-server agach-daemon
	rm -rf internal/server/ux/dist

docker:
	docker build -t agach-mcp .

docker_build:
	docker build -f Dockerfile.local --output type=local,dest=. .
