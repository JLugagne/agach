.PHONY: generate build run dev test clean docker docker_build

BINARY  := agach-server
GO_TAGS := sqlite_fts5

generate:
	cd ux && npm run build

build: generate
	CGO_ENABLED=1 go build -tags $(GO_TAGS) -o $(BINARY) .

run: build
	./$(BINARY)

dev: generate
	CGO_ENABLED=1 go run -tags $(GO_TAGS) .

test:
	CGO_ENABLED=1 go test -tags $(GO_TAGS) -race -failfast ./...

clean:
	rm -f $(BINARY)
	rm -rf ux/dist

docker:
	docker build -t agach-mcp .

docker_build:
	docker build -f Dockerfile.local --output type=local,dest=. .
