# Architecture

remote-cli has three components:

1. Relay
2. Agent
3. PWA

Claude Code runs as a local subprocess on each agent machine.

## Topology

```text
Phone / PWA  ----\
                  \
Laptop agent  -----\
Desktop agent ------> Relay
Pi agent      -----/
```

All connections are outbound to the relay.

## Relay

The relay is a Go server.

It handles:

- HTTP API routes
- WebSocket connections
- admin login
- JWT validation
- pairing codes
- device registry
- offline/online/busy presence
- session routing
- SQLite persistence
- serving the PWA

Persistent state:

- admin password hash
- device records
- device token hashes
- last-seen timestamps

In-memory state:

- active phone connections
- active agent connections
- session-to-device mappings
- session-to-phone-owner mappings
- tool-use-to-session mappings
- busy device IDs
- pairing codes

## Agent

The agent is a Go CLI.

Commands:

```bash
remote-cli pair --relay <url>
remote-cli pair --relay <url>
remote-cli run
remote-cli status
remote-cli unpair
remote-cli service install
remote-cli service uninstall
remote-cli service start
remote-cli service stop
remote-cli service logs
```

The agent stores config at:

```text
~/.config/remote-cli/config.toml
```

When running, it:

1. opens a WebSocket to the relay
2. sends `device.register`
3. waits for session messages
4. starts Claude Code when requested
5. streams Claude output back to the relay

## PWA

The PWA is a React/Vite app served by the relay.

It handles:

- login
- WebSocket auth
- device list
- add device / pairing
- chat view
- streamed assistant output
- tool event display

In production, users open the relay URL in a browser. There is no separate PWA deployment.

## Pairing Flow

```text
Agent -> Relay: open /ws/agent
Relay -> Agent: connected(connection_id)
Agent -> Relay: POST /api/pair/request(connection_id)
Relay -> Agent: pair.code(code, url)
Phone -> Relay: POST /api/pair/redeem(code)
Relay -> Agent: pair.complete(device_id, device_token)
Agent: save config
```

Pairing codes:

- are 6 digits
- expire after 5 minutes
- are single-use
- are bound to the requesting agent WebSocket connection

## Session Flow

```text
PWA -> Relay: session.start(device_id)
Relay: reject if device is offline or busy
Relay: create session_id
Relay: map session_id -> device_id
Relay: mark device busy
Relay -> Agent: session.start(session_id)
Agent: spawn claude
Agent -> Relay -> PWA: session.started
PWA -> Relay -> Agent: message.user
Agent -> Claude: stdin stream-json message
Claude -> Agent: stdout stream-json events
Agent -> Relay -> PWA: message.assistant_chunk
```

When the session ends, the relay removes the session mapping, marks the device online if the agent is still connected, and forwards `session.ended` to the phone.

## Multi-Device Routing

Each agent registers with a unique `device_id`.

The relay stores:

```text
device_id -> active agent connection
session_id -> device_id
```

When the PWA sends a user message, it includes only `session_id`. The relay uses the session map to find the right device and forwards the message to that agent.

## Claude Integration

The agent starts:

```bash
claude \
  --input-format stream-json \
  --output-format stream-json \
  --verbose \
  --include-partial-messages \
  --no-session-persistence \
  --permission-prompt-tool mcp__approval__request_permission \
  --mcp-config /tmp/remote-cli-mcp-<session-id>.json
```

Claude emits JSON lines. The agent parses those lines and converts them to remote-cli protocol messages.

For partial assistant messages, Claude emits cumulative text. The agent tracks the previous text per message ID and forwards only the new delta.

## Tool Approval

When Claude wants to run a tool it requires permission for, it calls the `request_permission` MCP tool instead of prompting the terminal.

The MCP server is a subprocess of the agent binary (`remote-cli mcp-server --socket <path>`). It connects to a Unix socket (the permission bridge) running inside the agent.

Flow:

```text
Claude -> MCP server subprocess: tools/call request_permission
MCP server -> Unix socket (bridge): bridgeRequest{tool_use_id, tool_name, tool_input}
Bridge -> Relay -> PWA: tool_use.request(awaiting_approval=true)
PWA: show Allow / Deny buttons
User taps Allow  -> PWA -> Relay -> Agent: tool_use.approve
User taps Deny   -> PWA -> Relay -> Agent: tool_use.deny
Bridge -> MCP server: bridgeResponse{allow: true/false}
MCP server -> Claude: {"behavior":"allow"} or {"behavior":"deny","message":"..."}
```

If the bridge fails to start (e.g. socket error), the session falls back to auto-approval.

## Disconnects

Phone disconnect:

- PWA reconnects automatically.
- Current in-memory UI state may be lost.

Agent disconnect:

- relay marks device offline
- active sessions for that device are ended
- agent reconnects when `remote-cli run` is still active
- if the agent is installed as a service, the service manager restarts it after failures

Relay restart:

- persistent device/admin state survives
- live sessions, pairing codes, and WebSocket connections are lost

## Current Limitations

- No persisted chat history.
- No end-to-end encryption.
- One active session per agent.
- No `remote-cli service status` command yet.
