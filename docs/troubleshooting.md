# Troubleshooting

Start with the relay URL and the agent process. Most early failures are one of:

- the phone or agent cannot reach the relay URL
- the device was paired but `remote-cli run` is not running
- Claude Code is not working locally on the agent machine

## Device Shows Offline After Pairing

Pairing only writes credentials to the agent config. It does not keep the agent online.

On the agent machine:

```bash
remote-cli status
remote-cli run
```

Keep `remote-cli run` running. The device should turn online in the PWA within a few seconds.

## QR Code Or Pairing URL Shows `localhost`

`localhost` only works from the same machine. If you scan a QR code containing `http://localhost:8080`, your phone tries to connect to itself, not your laptop.

Set `RELAY_URL` to a LAN IP, Tailscale hostname, or public HTTPS domain before starting the relay.

See [choosing-relay-url.md](choosing-relay-url.md).

## Phone Cannot Reach Relay

Symptoms:

- PWA does not load.
- Login says it cannot reach the relay.
- Reconnecting banner stays visible.

Check from the phone browser:

```text
http://YOUR_RELAY_URL
```

If the page does not load, fix networking first.

Common causes:

- relay URL is `localhost`
- phone is on cellular but relay is LAN-only
- firewall blocks port 8080
- relay host is asleep
- Docker container is not running
- reverse proxy is not forwarding WebSockets

For Docker:

```bash
docker compose ps
docker compose logs relay
```

## Device Shows Online But No Response To Messages

First check whether Claude works on the agent machine:

```bash
which claude
claude --version
claude --print "Reply with OK"
```

If Claude says you are not logged in, authenticate Claude Code on that machine.

If Claude says you are rate-limited, wait until the reset time and try again.

Then watch the terminal running:

```bash
remote-cli run
```

Useful logs include:

```text
session start: <session-id>
session <id>: claude pid <pid> started
session <id>: claude init
session <id>: turn complete
session <id>: claude stderr: ...
```

If no session logs appear after tapping a device, the relay may not be routing to an active agent connection.

## Agent Online, Then Suddenly Offline

The relay marks a device online only while its agent WebSocket is connected.

Common causes:

- the `remote-cli run` process exited
- laptop/desktop/Pi went to sleep
- relay host restarted
- network changed
- relay URL became unreachable
- firewall/VPN changed

Restart the agent:

```bash
remote-cli run
```

If it reconnects and then drops again, check the terminal logs and relay logs.

## Tapping A Device Does Nothing Or Session Never Starts

Check that the agent is currently connected to the relay.

On the agent machine:

```bash
remote-cli status
remote-cli run
```

Check the stored relay URL:

```bash
cat ~/.config/remote-cli/config.toml
```

If the relay URL is wrong:

```bash
remote-cli unpair
remote-cli pair --relay http://CORRECT_RELAY_URL
remote-cli run
```

## Claude Auth Or Rate Limit Errors

The agent uses Claude Code locally. Each agent machine needs its own working Claude Code setup.

Verify on the agent machine:

```bash
claude --print "Reply with OK"
```

If this fails, remote-cli cannot get a response from that machine either.

Common fixes:

- run the Claude Code login/auth flow
- wait for rate limits to reset
- confirm `claude` is on the `PATH` used by `remote-cli run`

## Phone Cannot Scan QR Code

Camera-based QR scanning may require HTTPS depending on browser and platform.

Fallback:

- tap manual code entry in the PWA
- enter the 6-digit code printed in the agent terminal

If the code expired, rerun:

```bash
remote-cli pair --relay <relay-url>
```

## Relay Behind Reverse Proxy: WebSocket Fails

The relay uses WebSockets for both phone and agent connections.

For nginx, include:

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

Caddy usually handles this automatically:

```caddy
relay.example.com {
    reverse_proxy localhost:8080
}
```

## Reset Pairing On An Agent

On the agent machine:

```bash
remote-cli unpair
remote-cli pair --relay <relay-url>
```

## Reset Relay State

For local development only, stop the relay and delete the SQLite database.

Warning: this removes admin and device state.

```bash
docker compose down
# remove the relay data volume or relay.db, depending on how you run it
docker compose up -d
```

There is no non-destructive admin password reset command yet.

## What To Include In Bug Reports

Please include:

- OS and architecture of relay and agent machines
- relay deployment mode: Docker or source
- relay URL shape: LAN IP, Tailscale, public domain, etc.
- whether the PWA shows the device online
- output of `remote-cli status`
- whether `claude --print "Reply with OK"` works on the agent machine
- relevant agent logs from `remote-cli run`
- relevant relay logs

Do not include tokens, JWTs, or private credentials.
