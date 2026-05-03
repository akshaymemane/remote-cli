.PHONY: relay agent test build-agent build-relay

relay:
	go run ./cmd/relay

agent:
	go run ./cmd/agent

test:
	go test ./...

build-relay:
	go build -o dist/relay ./cmd/relay

build-agent:
	mkdir -p dist
	GOOS=linux  GOARCH=amd64 go build -o dist/agent-linux-amd64   ./cmd/agent
	GOOS=linux  GOARCH=arm64 go build -o dist/agent-linux-arm64   ./cmd/agent
	GOOS=darwin GOARCH=amd64 go build -o dist/agent-darwin-amd64  ./cmd/agent
	GOOS=darwin GOARCH=arm64 go build -o dist/agent-darwin-arm64  ./cmd/agent
