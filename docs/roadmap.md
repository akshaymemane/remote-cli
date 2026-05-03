# Roadmap

remote-cli is in alpha. The roadmap is intentionally practical: make one complete user journey reliable before adding larger features.

## Milestone 1: Private Alpha

Goal: dogfood across a laptop, desktop, and Raspberry Pi.

Required:

- reliable relay startup
- reliable PWA login
- reliable pairing
- clear relay URL docs
- agent foreground run
- successful prompt/stream response
- clear Claude auth/rate-limit/spawn errors
- troubleshooting docs

## Milestone 2: Public Alpha

Goal: technical users can try it from GitHub without hand-holding.

Required:

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

## Milestone 3: Install Experience

Goal: reduce setup friction.

Planned:

- one-line agent installer
- clearer Docker deployment docs
- systemd user service helper
- launchd docs/helper
- `remote-cli logs`
- `remote-cli service status`

## Milestone 4: Reliability

Goal: make failures obvious and recoverable.

Planned:

- fail session start if agent is offline
- show actionable PWA errors
- better agent/relay logs
- reconnect/session cleanup behavior
- device delete/revoke
- admin password reset path

## Milestone 5: Product Depth

Possible:

- persisted chat history
- better multi-device switching
- online idle/busy/offline presence
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
