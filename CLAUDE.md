# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

**Multi-Device Claude Code Controller** — control multiple Claude Code sessions on different machines from a mobile PWA. Three components:

- **Agent** (`cmd/agent`, Go) — runs on each user machine; spawns real `claude` CLI sessions via stdin/stdout JSON-line protocol
- **Relay** (`cmd/relay`, Go) — WebSocket broker between phone and agents; SQLite state
- **PWA** (`pwa/`, React + Vite + TypeScript) — mobile interface served by the relay

## Commands

```bash
# Relay
go run ./cmd/relay                     # first run prompts for admin password
RELAY_ADDR=:9090 go run ./cmd/relay    # custom port

# Agent
go run ./cmd/agent pair --relay <url>  # pair this machine
go run ./cmd/agent run                 # start agent loop
go run ./cmd/agent status              # show config
go run ./cmd/agent unpair              # remove config

# PWA
cd pwa && npm install && npm run dev   # dev server on :5173 (set VITE_RELAY_URL)
cd pwa && npm run build                # production build → pwa/dist/

# Full stack (relay serves PWA from pwa/dist/)
cd pwa && npm run build && cd .. && go run ./cmd/relay

# Build binaries
make build-relay                       # ./dist/relay (4 platforms)
make build-agent                       # ./dist/agent-* (4 platforms)
go test ./...

# Docker
docker compose up -d                   # relay + PWA on :8080
```

## Key environment variables (relay)

| Var | Default | Purpose |
|-----|---------|---------|
| `RELAY_ADDR` | `:8080` | Listen address |
| `RELAY_DB` | `relay.db` | SQLite file path |
| `RELAY_JWT_SECRET` | `change-me-...` | JWT signing key |
| `RELAY_URL` | `http://localhost:8080` | Public base URL (used in QR code) |
| `RELAY_STATIC_DIR` | `pwa/dist` | PWA static files; empty = disable |

## Architecture

**Go module**: `remote-cli`. Packages under `internal/relay/`, `internal/agent/`, `internal/protocol/`.

**Claude sessions**: Agent spawns `claude --print --input-format stream-json --output-format stream-json --verbose --include-partial-messages --no-session-persistence`. The process stays alive across turns. `--include-partial-messages` yields cumulative text — agent diffs `lastText[msgID]` and sends only new chars as `message.assistant_chunk`.

**Hub pattern**: `internal/relay/hub.go` — `conn` structs with buffered `send chan []byte` + dedicated `writePump` goroutine per connection.

**Session lifecycle**: phone sends `session.start` → relay assigns sessionID → agent spawns claude process → agent sends `session.started` → bidirectional message flow → `session.end` or disconnect.

**Tool approval**: `tool_use.request` (agent→relay→phone), phone sends `tool_use.approve/deny`, relay looks up sessionID via `toolSessions` map, forwards to agent.

**Pairing**: agent calls `POST /api/pair/request` (no auth needed) → relay sends pair code via WS → phone calls `POST /api/pair/redeem` (JWT required) → relay sends device credentials back to agent WS.

**PWA**: SPA served from relay. Unknown paths fall back to `index.html`. `VITE_RELAY_URL` not needed in production (defaults to `window.location.origin`).

## WebSocket protocol (all frames JSON with `type` field)

```
phone → relay: client.auth, session.start, session.end, message.user, tool_use.approve, tool_use.deny
relay → phone: device.list, device.update, session.state, session.started, session.ended,
                message.assistant_chunk, tool_use.request, tool_use.result, error
agent → relay: device.register, device.heartbeat, session.started, session.ended,
                message.assistant_chunk, tool_use.request, tool_use.result, error
relay → agent: connected, session.start, session.end, message.user, tool_use.approve, tool_use.deny, pair.code, pair.complete
```

## Security

- Phone↔Relay: admin password → 24h HS256 JWT
- Agent↔Relay: 256-bit hex device token (bcrypt-hashed at rest; plaintext in `~/.config/remote-cli/config.toml` mode 0600)
- No E2E encryption — relay is self-hosted by the same person who owns the machines
