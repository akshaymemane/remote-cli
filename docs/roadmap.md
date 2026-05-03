# Roadmap

remote-cli is a beta-candidate, self-hosted project. The roadmap is intentionally practical: keep the complete user journey reliable before adding larger features.

## Milestone 1: Private Alpha

Goal: dogfood across a laptop, desktop, and Raspberry Pi.

Done:

- reliable relay startup
- reliable PWA login
- reliable pairing
- clear relay URL docs
- agent foreground run
- successful prompt/stream response
- clear Claude auth/rate-limit/spawn errors
- troubleshooting docs

## Milestone 2: Public Beta

Goal: technical users can try it from GitHub without hand-holding.

Required before tagging:

- public README
- license
- security docs
- Docker relay quickstart
- release binaries for agents
- checksums
- basic CI
- basic release workflow
- known limitations documented
- no committed local binaries, DBs, or dependency folders
- final smoke test from a clean checkout

## Milestone 3: Install Experience

Goal: reduce setup friction.

Partly done:

- release binary workflow
- Linux systemd user service helper
- macOS launchd service helper
- `remote-cli service logs`

Planned:

- one-line agent installer
- clearer Docker deployment docs
- `remote-cli service status`

## Milestone 4: Reliability

Goal: make failures obvious and recoverable.

Done:

- fail session start if agent is offline
- reject session start if the device is already busy
- show actionable PWA errors
- reconnect/session cleanup behavior
- device delete/revoke

Planned:

- better agent/relay logs
- admin password reset path

## Milestone 5: Product Depth

Possible:

- persisted chat history
- better multi-device switching
- project/workdir selection
- richer tool display
- notification support

## Milestone 6: Security Hardening

Possible:

- stricter production defaults
- optional origin restrictions
- login rate limiting
- token revocation
- E2E encryption research
- security review

## Not In v1

- hosted relay service
- multi-user/team model
- Windows agent support
- attaching to existing Claude TUI sessions
- native mobile app
