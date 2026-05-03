# Contributing

Thanks for considering a contribution. remote-cli is an early beta-candidate project, so the highest-impact work right now is making the core user journey reliable and understandable.

## Project Status

Current shape:

- Relay, agent, and PWA exist in a monorepo.
- Pairing works in the happy path.
- Agents connect outbound to the relay.
- The agent can spawn Claude Code using stream-json mode.
- The PWA has login, device list, pairing, and chat views.
- Basic macOS launchd and Linux systemd user-service helpers exist for the agent.

Known gaps:

- Phone-side tool approval is not a supported permission boundary yet.
- Chat history is not persisted yet.
- Some failure cases still need real-world testing across networks, browsers, and Claude Code states.

## Repository Layout

```text
cmd/
  relay/       Relay server entrypoint
  agent/       Agent CLI entrypoint
internal/
  relay/       HTTP routes, WebSocket hub, auth, pairing, SQLite
  agent/       Relay client, config, pairing, Claude session management
  protocol/    Shared JSON message types
pwa/           React + Vite + TypeScript PWA
docs/          User and developer documentation
codex/         Planning/audit notes
```

## Local Development

Relay:

```bash
go run ./cmd/relay
```

On first run, the relay prompts for an admin password unless `RELAY_ADMIN_PASSWORD` is set.

Agent:

```bash
go run ./cmd/agent pair --relay http://localhost:8080
go run ./cmd/agent run
go run ./cmd/agent status
go run ./cmd/agent unpair
```

PWA:

```bash
cd pwa
npm install
VITE_RELAY_URL=http://localhost:8080 npm run dev
```

The PWA dev server does not proxy relay traffic by itself; `VITE_RELAY_URL` tells it which relay to talk to.

Full stack with relay serving the built PWA:

```bash
cd pwa
npm run build
cd ..
go run ./cmd/relay
```

More detail is in [docs/development.md](docs/development.md).

## Testing

Go:

```bash
go test ./...
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

## Code Style

- Use `gofmt` for Go code.
- Follow the existing TypeScript style in `pwa/`.
- Keep changes focused.
- Prefer small PRs with one clear concern.
- Add comments only where they explain non-obvious reasoning.

## Useful Architecture Notes

Message flow:

```text
PWA -> Relay: session.start
Relay -> Agent: session.start
Agent -> Claude: spawn process
PWA -> Relay -> Agent: message.user
Agent -> Relay -> PWA: message.assistant_chunk
```

The relay tracks:

- active agent connections by device ID
- session ID to device ID mappings
- pending tool-use ID to session ID mappings

The agent runs Claude Code with:

```bash
claude --print \
  --input-format stream-json \
  --output-format stream-json \
  --verbose \
  --include-partial-messages \
  --no-session-persistence
```

## Opening Issues

For bugs, include:

- OS and architecture
- relay deployment mode: Docker or source
- relay URL shape: LAN IP, Tailscale, public domain, etc.
- browser/device used for the PWA
- whether the device appears online
- whether `claude --print "Reply OK"` works on the agent machine
- relevant relay and agent logs

Do not include secrets, device tokens, JWTs, or private relay URLs.

## Pull Requests

Please include:

- what changed
- why it changed
- how you tested it
- screenshots for PWA changes
- any security impact
- docs updates when behavior changes

Security-sensitive changes should be extra explicit about trust boundaries.
