# Choosing Your Relay URL

The relay URL is one of the most important setup choices in remote-cli.

Every phone and every agent machine must be able to reach the same relay URL.

## What Uses The Relay URL?

The relay URL is used by:

1. The PWA in your phone browser.
2. The pairing QR code shown by the agent.
3. Agent WebSocket connections after pairing.
4. Phone WebSocket connections after login.

If this URL is wrong, pairing may fail, devices may show offline, or chat messages may appear to go nowhere.

## Why `localhost` Usually Fails

`localhost` always means "this same machine."

If the relay runs on your laptop:

- `localhost` on the laptop means the laptop.
- `localhost` on your phone means the phone.
- `localhost` on your desktop means the desktop.
- `localhost` on your Raspberry Pi means the Pi.

So this is usually wrong for multi-device use:

```bash
RELAY_URL=http://localhost:8080
```

Use a URL that resolves correctly from the phone and all agent machines.

## Option 1: Same LAN

Use this for first tests at home or on the same Wi-Fi.

Find the relay machine's LAN IP.

macOS:

```bash
ipconfig getifaddr en0
```

Linux:

```bash
ip route get 1 | awk '{print $7; exit}'
```

Set:

```bash
RELAY_URL=http://192.168.1.10:8080
```

Replace `192.168.1.10` with your actual relay machine IP.

Works when:

- phone is on the same network
- agent machines are on the same network
- firewall allows inbound traffic to the relay port
- the relay host does not sleep

## Option 2: Tailscale

This is a good private remote-access option.

Install Tailscale on:

- relay host
- agent machines
- phone

Then use the relay host's Tailscale IP or MagicDNS name:

```bash
RELAY_URL=http://100.x.y.z:8080
```

or:

```bash
RELAY_URL=http://relay-host.tailnet-name.ts.net:8080
```

This avoids exposing the relay publicly.

## Option 3: Public HTTPS Domain

Use this for production-style deployment.

```bash
RELAY_URL=https://relay.example.com
```

Put the relay behind a reverse proxy with TLS.

Caddy example:

```caddy
relay.example.com {
    reverse_proxy localhost:8080
}
```

Caddy handles WebSocket upgrades automatically.

For nginx, make sure WebSocket upgrade headers are forwarded:

```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
proxy_set_header Host $host;
```

## Quick Verification

Before pairing agents, test the relay URL.

From your phone browser:

```text
http://192.168.1.10:8080
```

From each agent machine:

```bash
curl http://192.168.1.10:8080
```

If these fail, fix networking before debugging remote-cli.

## Recommended Defaults By Scenario

| Scenario | Relay URL |
| --- | --- |
| Testing on one machine | `http://localhost:8080` |
| Phone + devices on same Wi-Fi | `http://<LAN-IP>:8080` |
| Private remote access | `http://<tailscale-name-or-ip>:8080` |
| Public deployment | `https://relay.example.com` |

## Rule Of Thumb

If your phone cannot open the relay URL in a browser, the PWA and pairing flow will not work.
