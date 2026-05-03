# Protocol

remote-cli uses JSON messages over WebSockets.

Every message has a `type` field.

## WebSocket Endpoints

```text
/ws/phone
/ws/agent
```

Phone connections must send `client.auth` as the first message.

Agent connections receive `connected` first, then either request pairing or send `device.register`.

## HTTP Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/auth/login` | Exchange admin password for JWT. |
| `POST` | `/api/pair/request` | Agent requests a pairing code. |
| `POST` | `/api/pair/redeem` | Authenticated phone redeems a pairing code. |
| `GET` | `/api/debug/devices` | Development/debug device listing. |

## Pairing Messages

### `connected`

Relay to agent.

```json
{
  "type": "connected",
  "connection_id": "..."
}
```

### `pair.code`

Relay to pairing agent.

```json
{
  "type": "pair.code",
  "code": "123456",
  "url": "http://relay/pair/123456"
}
```

### `pair.complete`

Relay to pairing agent.

```json
{
  "type": "pair.complete",
  "device_id": "...",
  "device_token": "...",
  "device_name": "..."
}
```

## Phone To Relay

### `client.auth`

```json
{
  "type": "client.auth",
  "token": "jwt..."
}
```

### `device.list`

```json
{
  "type": "device.list"
}
```

### `session.start`

```json
{
  "type": "session.start",
  "device_id": "..."
}
```

### `session.end`

```json
{
  "type": "session.end",
  "device_id": "...",
  "session_id": "..."
}
```

### `message.user`

```json
{
  "type": "message.user",
  "session_id": "...",
  "content": "..."
}
```

### `tool_use.approve`

```json
{
  "type": "tool_use.approve",
  "tool_use_id": "..."
}
```

Current alpha note: approve/deny messages exist in the protocol, but phone approval is not a supported end-to-end permission path yet.

### `tool_use.deny`

```json
{
  "type": "tool_use.deny",
  "tool_use_id": "...",
  "reason": "..."
}
```

## Agent To Relay

### `device.register`

```json
{
  "type": "device.register",
  "device_id": "...",
  "device_token": "..."
}
```

### `device.heartbeat`

```json
{
  "type": "device.heartbeat"
}
```

### `session.started`

```json
{
  "type": "session.started",
  "session_id": "..."
}
```

### `session.ended`

```json
{
  "type": "session.ended",
  "session_id": "...",
  "reason": "..."
}
```

### `message.assistant_chunk`

```json
{
  "type": "message.assistant_chunk",
  "session_id": "...",
  "content_block": {
    "type": "text",
    "text": "Hello"
  },
  "index": 0
}
```

### `tool_use.request`

```json
{
  "type": "tool_use.request",
  "session_id": "...",
  "tool_use_id": "...",
  "tool_name": "Bash",
  "tool_input": {
    "command": "ls"
  }
}
```

### `tool_use.result`

```json
{
  "type": "tool_use.result",
  "session_id": "...",
  "tool_use_id": "...",
  "result": "..."
}
```

### `error`

```json
{
  "type": "error",
  "session_id": "...",
  "code": "spawn_failed",
  "message": "..."
}
```

## Relay To Phone

### `device.list`

```json
{
  "type": "device.list",
  "devices": [
    {
      "id": "...",
      "name": "laptop",
      "status": "online",
      "last_seen": 1777787729
    }
  ]
}
```

### `device.update`

```json
{
  "type": "device.update",
  "device_id": "...",
  "status": "offline",
  "last_seen": 1777787729
}
```

### `session.state`

```json
{
  "type": "session.state",
  "device_id": "...",
  "session_id": "...",
  "status": "starting"
}
```

Relay also forwards agent messages such as `session.started`, `session.ended`, `message.assistant_chunk`, `tool_use.request`, `tool_use.result`, and `error`.

## Relay To Agent

Relay sends:

- `session.start`
- `session.end`
- `message.user`
- `tool_use.approve`
- `tool_use.deny`

## Versioning

There is no protocol version field yet. Before stable release, add a compatibility/versioning policy if third-party clients are expected.
