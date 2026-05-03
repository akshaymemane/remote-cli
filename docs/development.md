# Development

This guide is for working on remote-cli locally.

## Prerequisites

- Go 1.23+
- Node.js 20+
- npm
- SQLite CGO build support
- Docker, if testing the container
- Claude Code, if testing real agent sessions

## Run Relay

```bash
go run ./cmd/relay
```

Useful env vars:

```bash
RELAY_ADDR=:8080
RELAY_DB=relay.db
RELAY_URL=http://localhost:8080
RELAY_STATIC_DIR=pwa/dist
RELAY_JWT_SECRET=dev-secret-change-me
RELAY_ADMIN_PASSWORD=dev-password
```

## Run PWA In Dev Mode

```bash
cd pwa
npm install
VITE_RELAY_URL=http://localhost:8080 npm run dev
```

Open:

```text
http://localhost:5173
```

## Run Full Stack Locally

Build PWA:

```bash
cd pwa
npm run build
cd ..
```

Run relay:

```bash
go run ./cmd/relay
```

Open:

```text
http://localhost:8080
```

## Run Agent

Pair:

```bash
go run ./cmd/agent pair --relay http://localhost:8080
```

Run:

```bash
go run ./cmd/agent run
```

Run after pairing:

```bash
go run ./cmd/agent pair --relay http://localhost:8080 --run
```

Status:

```bash
go run ./cmd/agent status
```

Unpair:

```bash
go run ./cmd/agent unpair
```

Service helpers are available in built binaries on macOS and Linux:

```bash
remote-cli service install
remote-cli service logs
```

## Tests And Builds

Go:

```bash
go test ./...
go build ./cmd/relay
go build ./cmd/agent
```

PWA:

```bash
cd pwa
npm run build
npm run lint
```

Docker:

```bash
docker compose build
```

## Debugging Message Flow

Watch relay logs for:

```text
phone connected
agent registered
session started
session ended
```

Watch agent logs for:

```text
connected as
session start
claude pid
claude init
turn complete
claude stderr
```

If no response appears in the PWA, first verify:

```bash
claude --print "Reply with OK"
```

on the agent machine.

## Local Data

Relay DB:

```text
relay.db
```

Agent config:

```text
~/.config/remote-cli/config.toml
```

For development, deleting these resets state. Do not delete production state without a backup.

## Release Workflow

The release workflow is intended to run on tags:

```text
v*
```

It builds agent binaries and publishes a relay Docker image.

Before using it publicly, confirm repository owner/name, GHCR permissions, and release asset names.
