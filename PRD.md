# PRD: Multi-Device Claude Code Controller

**Status:** Draft v1
**Last updated:** May 2026
**Owner:** TBD
**License:** TBD (recommend MIT or Apache 2.0 before first public commit)

---

## 1. Summary

An open-source system for controlling multiple Claude Code sessions running on different machines (laptops, desktops, Raspberry Pis) from a single mobile interface. Modeled on a messaging-app metaphor: each machine appears as a "contact," each session as a "chat."

The system consists of three components:

- A lightweight **agent** that runs on each machine and hosts Claude Code sessions on demand.
- A **relay** server that brokers messages between phones and agents and tracks device presence.
- A **mobile PWA** that provides the user-facing interface.

The v1 goal is a thin pipe: the agent does not impose its own permission policy or feature set on top of Claude Code. It exposes Claude Code as it is, accessible from a phone.

---

## 2. Problem

Users running Claude Code across multiple machines have no unified way to interact with those sessions from a phone. The current options are unsatisfactory for fleet use:

- **Native Claude Code Remote Control** supports per-machine remote access via `claude.ai/code`, but switching between sessions on a phone is awkward, and there is no fleet-level view.
- **SSH + tmux** works but bypasses the structured tool-use and approval model.
- **Claude Code on the web** runs in cloud sandboxes and cannot access a user's local files, MCP servers, or project configuration.

The result: users either stay tethered to their workstations or lose access to in-progress sessions when they step away.

## 3. Goals and non-goals

### Goals

- Single mobile interface ("inbox") showing all of a user's machines and their online status.
- Tap-into-chat experience: select a device, start a Claude Code session, send prompts, approve tool calls, see streamed output.
- Fast switching between devices via a back-to-list gesture (WhatsApp-style).
- Self-hostable with minimal setup. A single `docker compose up` should bring up the relay; a single shell command should install the agent.
- Pairing via QR code rendered in the terminal; numeric fallback for any environment where QR scanning is unavailable.
- Open-source, contributor-friendly architecture.

### Non-goals (v1)

- Push notifications.
- Multi-project chats per device. (One device, one chat.)
- Per-device permission policies or managed approval rules.
- iOS native app (PWA only).
- Hosted relay run by the project. (Self-host only.)
- Multi-user / team support.
- End-to-end encryption between phone and agent. (Self-hosters control both endpoints; TLS per hop is sufficient.)
- Windows agent if it adds significant packaging complexity. (Linux + macOS covers the stated use case.)
- Attaching to pre-existing local CLI sessions. The agent always spawns fresh sessions through the SDK.
- Message search, chat export, advanced syntax highlighting beyond what a markdown renderer provides.
- CLI-as-controller (using one Claude Code instance to control another).

## 4. User experience

### 4.1 First-time setup

1. User runs the relay on a server, VPS, or home machine: `docker compose up -d`.
2. User opens the mobile PWA and signs in with the relay's admin credential.
3. User installs the agent on a machine: `curl -sSL get.<project>.dev | sh`.
4. User runs `<project> pair` on the machine. A QR code and a 6-digit numeric code appear in the terminal.
5. User opens the PWA, taps **Add device**, scans the QR code (or types the numeric code).
6. The device appears in the device list within ~2 seconds. The terminal prints `Paired as <hostname>` and exits.

Total time, post-relay: under 60 seconds per machine.

### 4.2 Daily use

1. User opens the PWA. Devices list shows each registered machine with a presence indicator: **online + idle**, **online + busy**, or **offline** (with last-seen timestamp).
2. User taps a device. If no chat is active, a new session starts. If one is active, the existing chat opens.
3. User types a prompt. The agent spawns a Claude Code session via the SDK and streams the response back.
4. When Claude requests a tool call, the chat shows an inline approval card with **Approve** / **Deny** buttons.
5. User can switch devices at any time via the back gesture. In-flight sessions on other devices continue running.
6. User ends the chat explicitly via an **End session** action, or after an idle timeout (default: 1 hour) the agent terminates the session.

### 4.3 Failure cases the UI must handle

- **Agent goes offline mid-session.** Chat shows a clear banner: `<device> went offline. Session ended.` Subsequent prompts are disabled until the device reconnects.
- **Phone loses connectivity.** Phone reconnects to the relay on resume and pulls the message backlog. The agent and its session continue running unaffected.
- **Tool call mid-execution when WebSocket dies.** Agent continues the tool call locally; result is delivered when the connection recovers.
- **Pairing code expires before scan.** PWA shows `Code expired. Re-run pair on the machine.`

## 5. Architecture

### 5.1 Components

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│             │ HTTPS/  │              │ HTTPS/  │             │
│  Mobile PWA │  WSS    │    Relay     │  WSS    │   Agent     │
│             │ ──────► │   (server)   │ ◄────── │ (machine)   │
│             │         │              │         │             │
└─────────────┘         └──────────────┘         └─────────────┘
                                                        │
                                                        │ spawns
                                                        ▼
                                              ┌──────────────────┐
                                              │  Claude Code     │
                                              │  session via     │
                                              │  Agent SDK       │
                                              └──────────────────┘
```

All connections are outbound from the agent and the phone toward the relay. No inbound ports on user machines.

### 5.2 Agent

- **Language:** Go. Single static binary, trivial cross-compilation for `linux/amd64`, `linux/arm64`, `linux/armv7`, `darwin/amd64`, `darwin/arm64`.
- **Claude Code integration:** The Go agent spawns a bundled Node.js helper that uses the [Claude Agent SDK (TypeScript)](https://docs.claude.com/en/api/agent-sdk/overview). The helper exposes a stdin/stdout JSON-line protocol; the Go process pipes messages and tool-use events between the helper and the relay's WebSocket.
- **Lifecycle:** On startup, the agent connects to the relay over WebSocket and registers using its device credential. Between sessions, the agent stays connected (lightweight) but does not run a Claude Code session. When the relay sends `session.start`, the agent spawns the SDK helper. When the chat ends or times out, the agent terminates the helper subprocess.
- **Configuration:** `~/.config/<project>/config.toml`. Holds relay URL, device credential, and (optionally) device display name. Pairing writes this file; users do not edit it manually.
- **Service management:** Installed as a user-level systemd unit on Linux, a launchd agent on macOS. No root required.
- **Logs:** Available via `<project> logs` (which wraps `journalctl --user -u <project>` on Linux, equivalent on macOS).
- **Updates:** `<project> upgrade` fetches the latest release binary and replaces itself.

### 5.3 Relay

- **Language:** Go (same toolchain as the agent — simplifies builds and shared code).
- **Protocol:** WebSocket (over TLS). One connection per phone, one per agent.
- **State:** SQLite for persistent state (devices, credentials, last-seen). In-memory for live connections and pairing codes. SQLite chosen over Postgres for self-host simplicity — no separate DB to deploy.
- **Deployment:** A single `docker compose up` brings up the relay with Caddy as a TLS-terminating reverse proxy (automatic Let's Encrypt). Users supply a domain via env var; everything else is automatic.
- **Auth:** Single admin account, configured at first run. The admin logs into the PWA. Pairing endpoints require an admin session.
- **Pairing subsystem:** Generates short-lived codes (5-minute TTL) bound to the requesting agent's WebSocket connection. Codes are 6 numeric digits; QR encodes a URL of the form `https://<relay>/pair/<code>`.
- **Device tokens:** Long-lived random tokens (256-bit, hex-encoded). Stored hashed in the relay's DB, in plaintext in the agent's config file (treated as a credential).

### 5.4 Mobile PWA

- **Stack:** Suggested — React + Vite. Or any equivalent SPA framework. Chosen for ecosystem and contributor familiarity.
- **Distribution:** Served by the relay on the same domain. Users access it by opening the relay URL in a browser and adding to home screen.
- **Camera / QR scanning:** [`html5-qrcode`](https://github.com/mebjas/html5-qrcode) for the camera-based scanner. Requires HTTPS, which the relay provides by default via Caddy.
- **State:** All chat history is held by the relay (and persisted in SQLite). The PWA fetches on demand and updates over WebSocket. No client-side persistent store in v1.
- **Offline behavior:** Read-only when offline, showing the last-loaded state.

### 5.5 Why not...

- **Why not Tailscale-based peer-to-peer?** Requires users to install and configure a separate product, fragments the auth model, and reinvents the relay's coordination role. The relay-based design works for any network. Tailscale will be documented as an optional advanced deployment (relay-on-tailnet) but never required.
- **Why not embed Claude Code itself in the agent?** The Agent SDK is the supported programmatic surface; driving the `claude` TUI via a pty is fragile and not the intended interface.
- **Why not Postgres?** Self-host simplicity. SQLite handles the load comfortably for any single-user fleet.
- **Why not a native iOS app?** Apple developer fees, sideloading friction, and review overhead are out of scope for v1. PWA covers iOS and Android with one codebase.

## 6. Protocol specification

WebSocket frames carry JSON messages. Every message has a `type` field; additional fields depend on the type. Messages stream in both directions on each connection.

### 6.1 Agent → Relay

| Type                      | Purpose                                                    | Key fields                                             |
| ------------------------- | ---------------------------------------------------------- | ------------------------------------------------------ |
| `device.register`         | Authenticate after connect                                 | `device_token`                                         |
| `device.heartbeat`        | Optional keepalive (the WebSocket itself signals liveness) | none                                                   |
| `session.started`         | Confirm a session has spawned                              | `session_id`                                           |
| `session.ended`           | Confirm a session has terminated                           | `session_id`, `reason`                                 |
| `message.assistant_chunk` | Streamed model output                                      | `session_id`, `content_block`, `index`                 |
| `tool_use.request`        | Claude wants to call a tool                                | `session_id`, `tool_use_id`, `tool_name`, `tool_input` |
| `tool_use.result`         | Result of an executed tool call                            | `session_id`, `tool_use_id`, `result`                  |
| `error`                   | Something went wrong                                       | `session_id?`, `code`, `message`                       |

### 6.2 Phone → Relay

| Type               | Purpose                              | Key fields                |
| ------------------ | ------------------------------------ | ------------------------- |
| `client.auth`      | Authenticate with admin credentials  | `token`                   |
| `device.list`      | Request current device list          | none                      |
| `session.start`    | Open a session on a device           | `device_id`               |
| `session.end`      | Close the active session on a device | `device_id`, `session_id` |
| `message.user`     | Send a user prompt                   | `session_id`, `content`   |
| `tool_use.approve` | Approve a pending tool call          | `tool_use_id`             |
| `tool_use.deny`    | Deny a pending tool call             | `tool_use_id`, `reason?`  |

### 6.3 Relay → Phone

| Type                      | Purpose                            | Key fields                                 |
| ------------------------- | ---------------------------------- | ------------------------------------------ |
| `device.list`             | Current devices and presence       | `devices: [{id, name, status, last_seen}]` |
| `device.update`           | Single device state changed        | `device_id`, `status`, `last_seen`         |
| `session.state`           | Current session state for a device | `device_id`, `session_id?`, `status`       |
| `message.assistant_chunk` | Forwarded from agent               | (same as agent message)                    |
| `tool_use.request`        | Forwarded from agent               | (same as agent message)                    |
| `error`                   | Forwarded or relay-generated       | `code`, `message`                          |

### 6.4 Relay → Agent

| Type               | Purpose                      | Key fields                             |
| ------------------ | ---------------------------- | -------------------------------------- |
| `session.start`    | Spawn a Claude Code session  | `session_id`                           |
| `session.end`      | Terminate the active session | `session_id`, `reason`                 |
| `message.user`     | Forwarded user prompt        | `session_id`, `content`                |
| `tool_use.approve` | Forwarded approval           | `session_id`, `tool_use_id`            |
| `tool_use.deny`    | Forwarded denial             | `session_id`, `tool_use_id`, `reason?` |

### 6.5 Pairing (HTTP, not WebSocket)

| Method | Endpoint            | Caller | Purpose                                                                   |
| ------ | ------------------- | ------ | ------------------------------------------------------------------------- |
| `POST` | `/api/pair/request` | Agent  | Request a pairing code (must be on a registered WebSocket)                |
| `POST` | `/api/pair/redeem`  | Phone  | Redeem a code, receive device credential, agent receives it via WebSocket |

Pairing codes expire after 5 minutes. A code is bound to the WebSocket connection that requested it.

## 7. Presence model

Three states: **online + idle**, **online + busy**, **offline**.

- **Connection:** WebSocket itself is the heartbeat. If the connection drops, the relay marks the device offline.
- **Heartbeat fallback:** Optional `device.heartbeat` ping every 15 seconds. Mark offline after 45 seconds of silence (only applies if WebSocket-level keepalive is unreliable on a given network).
- **Busy:** Agent reports busy when a session is mid-response (model is generating, tool call is executing, or waiting for user approval).
- **Last seen:** Persisted on every disconnect. Surfaced in the UI for offline devices.

## 8. Security

- **Transport:** TLS on every hop. Caddy handles the public-facing hop; agents and phones validate certificates normally.
- **Auth:**
  - Phone → Relay: admin password (set at relay first run), exchanged for a session JWT.
  - Agent → Relay: device token (256-bit, generated during pairing). Token is hashed at rest on the relay.
- **Pairing:** Short-lived codes (5 min), single-use, bound to the requesting agent's connection. The redeemer must be an authenticated admin.
- **No inbound ports on user machines.** Only the relay needs a public address.
- **Scope of trust:** The relay sees all messages in plaintext (no E2E encryption in v1). This is acceptable because the v1 deployment model is self-hosted: the relay operator and the user are the same person.
- **Logging:** Relay logs metadata (connection events, errors) but not message content by default. Message content is persisted in SQLite for chat history.

## 9. Build plan

Four phases, each independently demoable. Vertical "build one feature end-to-end" is explicitly avoided in favor of horizontal layers.

### Phase 1 — Relay + agent skeleton

- Relay: WebSocket server, SQLite schema, admin auth, pairing endpoints, debug HTTP endpoint listing connected devices.
- Agent: WebSocket client, config file, register-and-heartbeat loop, `pair` subcommand with terminal QR rendering.
- **Demo:** Run `docker compose up`, run `agent pair`, redeem code via `curl`, see the device appear in the relay's debug endpoint.

### Phase 2 — End-to-end message flow with a fake Claude

- Define and implement the full message protocol (Section 6).
- Agent's "Claude" is an echo: any user message comes back as `assistant_chunk` events that spell the same text.
- **Demo:** Send a `message.user` via `curl`, see the echoed response stream back through the relay.

### Phase 3 — Real Claude Code session

- Replace the echo with the Node.js SDK helper.
- Implement session lifecycle (`session.start` / `session.end`), streaming, tool-use requests and approvals.
- Idle timeout (1 hour default).
- **Demo:** Send prompts via `curl`, get real Claude responses, approve tool calls via `curl`.

### Phase 4 — Mobile PWA

- Devices list with presence indicators.
- Chat view with streamed messages, tool-call approval cards, end-session button.
- Pairing flow with camera-based QR scanner and numeric fallback.
- Connection status banners.
- **Demo:** Install agent on three machines, control all three from a phone.

## 10. Open issues and v2 candidates

- **Push notifications** for tool-call approvals. Will likely require a backend for APNs/FCM.
- **Hosted relay** with end-to-end encryption (relay sees ciphertext only). Lets the project lower the setup bar without becoming a privacy custodian.
- **Per-device permission policies.** "Auto-approve reads on this Pi, ask every time on the work laptop."
- **Multi-project chats per device.**
- **CLI client.** A terminal-based controller for users who prefer it over the PWA.
- **Tailscale documentation.** A `docs/advanced/tailscale.md` guide for users who want to host the relay on a tailnet with no public exposure.
- **Windows agent.**
- **Multi-user / teams.** Likely needs a different auth model entirely.

## 11. Definition of done for v1

- Three machines (one Mac, one Linux laptop, one Raspberry Pi) can each run the agent and pair via QR code in under 60 seconds.
- The phone PWA shows all three devices with correct presence states.
- Tapping any device opens a chat that can send prompts, receive streamed responses, and handle tool-call approvals.
- Switching devices is instantaneous from the user's perspective.
- The relay can be deployed by a new user with `docker compose up` and a domain, no other configuration.
- README has a 60-second demo GIF, a three-command quickstart, and links to deeper docs.
- Repository has a license, a `CONTRIBUTING.md`, and a working CI pipeline that builds the agent for all supported platforms.
