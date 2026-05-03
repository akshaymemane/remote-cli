# Security

remote-cli is an early self-hosted project. Please read this before exposing a relay outside your private network.

## Trust Model

You run the relay on infrastructure you control. Agents and phones connect to that relay. In v1, the relay is trusted infrastructure and can see traffic contents.

There is currently **no end-to-end encryption** between phone and agent.

## Component Boundaries

| Path | Auth mechanism | Notes |
| --- | --- | --- |
| Phone/PWA -> Relay | Admin password exchanged for a 24-hour HS256 JWT | JWT is signed with `RELAY_JWT_SECRET`. |
| Agent -> Relay | Random device token | Token is bcrypt-hashed in SQLite and stored plaintext in the agent config. |
| Agent -> Claude Code | Local subprocess | Claude Code runs on the agent machine, not on the relay. |

Agent config is stored at:

```text
~/.config/remote-cli/config.toml
```

The config file is written with mode `0600`.

## What The Relay Can See

The relay can see:

- user prompts
- assistant responses
- tool names, inputs, and results
- device names
- online/offline presence
- session IDs and routing metadata

Do not run the relay on infrastructure you do not trust.

## Command Execution Risk

The relay does not directly open a shell on agent machines. However, the relay forwards prompts and session messages to agents, and the agent runs Claude Code locally. Depending on Claude Code settings and permission mode, prompts may cause Claude Code to use tools on the agent machine.

Treat relay admin access as sensitive. A compromised relay or admin session can influence what is sent to Claude Code on connected agents.

## Production Hardening

Before exposing the relay beyond localhost or a private LAN:

1. Use TLS.
   - Put the relay behind Caddy, nginx, Traefik, Tailscale, or another TLS-capable proxy.
   - Do not expose plain HTTP on the public internet.

2. Set a strong JWT secret.
   - Generate one with:

```bash
openssl rand -hex 32
```

3. Set a strong admin password.
   - Use `RELAY_ADMIN_PASSWORD` for non-interactive Docker setup or the first-run prompt for local source runs.

4. Use a trusted network.
   - For early alpha, Tailscale or a private LAN is safer than a public IP.

5. Protect the SQLite database.
   - It contains the admin password hash, device token hashes, device names, and last-seen data.

6. Keep the agent config private.
   - The local config contains the plaintext device token.

## Current Alpha Limitations

- No end-to-end encryption.
- No device revoke/delete command yet.
- No per-device permission policy.
- No multi-user/team auth model.
- No server-side JWT revocation.
- Tool approval from the phone is not a supported permission path yet.

## Reporting Vulnerabilities

Please report security issues privately.

Email: akshaymemane29@gmail.com

Please include:

- description of the issue
- steps to reproduce
- impact
- affected version or commit
- any suggested fix

Do not open a public GitHub issue for vulnerabilities.

## Supported Versions

This project is alpha. Only the latest `main` branch and latest tagged alpha release, once releases exist, are considered supported.
