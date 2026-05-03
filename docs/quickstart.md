# Quickstart

This guide gets one relay, one phone/PWA, and one agent machine working.

For multi-device setups, repeat the agent install, pairing, and `remote-cli run` steps on every machine.

## 0. Pick A Relay URL

Choose a URL reachable from:

- your phone
- the relay machine
- every agent machine

For same-Wi-Fi testing:

```bash
RELAY_URL=http://192.168.1.10:8080
```

For Tailscale:

```bash
RELAY_URL=http://relay-host.tailnet-name.ts.net:8080
```

For public deployment:

```bash
RELAY_URL=https://relay.example.com
```

Avoid `localhost` unless everything runs on the same machine.

See [choosing-relay-url.md](choosing-relay-url.md).

## 1. Start The Relay

Clone the repo:

```bash
git clone https://github.com/akshaymemane/remote-cli.git
cd remote-cli
cp .env.example .env
```

Edit `.env`:

```bash
RELAY_URL=http://192.168.1.10:8080
RELAY_JWT_SECRET=<generate-with-openssl-rand-hex-32>
RELAY_ADMIN_PASSWORD=<choose-a-password>
```

Generate a secret:

```bash
openssl rand -hex 32
```

Start:

```bash
docker compose up -d
```

Check logs:

```bash
docker compose logs relay
```

## 2. Open The PWA

On your phone, open the relay URL:

```text
http://192.168.1.10:8080
```

Sign in with `RELAY_ADMIN_PASSWORD`.

## 3. Build Or Install The Agent

Until public release binaries exist, build from source:

```bash
go build -o remote-cli ./cmd/agent
chmod +x remote-cli
```

Optionally place it on your `PATH`:

```bash
mkdir -p ~/.local/bin
mv remote-cli ~/.local/bin/
```

Confirm:

```bash
remote-cli status
```

If the shell cannot find `remote-cli`, use `./remote-cli` from the build directory or add `~/.local/bin` to your `PATH`.

## 4. Confirm Claude Works Locally

On the agent machine:

```bash
which claude
claude --version
claude --print "Reply with OK"
```

If Claude Code is not installed, not authenticated, or rate-limited, remote-cli will not be able to get a useful response from that machine.

## 5. Pair The Agent

On the agent machine:

```bash
remote-cli pair --relay http://192.168.1.10:8080
```

The terminal prints a QR code and a 6-digit code.

In the PWA:

1. Tap add device.
2. Scan the QR code or enter the code manually.
3. Wait for the terminal to say pairing succeeded.

## 6. Run The Agent

Pairing saves credentials. It does not keep the agent online.

Run:

```bash
remote-cli run
```

Keep this process running.

The PWA should show the device online.

## 7. Start A Session

In the PWA:

1. Tap the online device.
2. Wait for the session to start.
3. Send a prompt.

Claude runs on the selected agent machine, not on the relay.

## 8. Add More Machines

Repeat on each laptop, desktop, or Raspberry Pi:

```bash
remote-cli pair --relay http://192.168.1.10:8080
remote-cli run
```

Each machine appears as a separate device in the PWA.

## Next Steps

- [Relay Deployment](relay-deployment.md)
- [Agent Install](agent-install.md)
- [Service And Autostart](service-autostart.md)
- [Troubleshooting](troubleshooting.md)
