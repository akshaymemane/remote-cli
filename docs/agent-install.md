# Agent Install

Install the agent on every machine you want to control.

The agent:

- pairs with the relay
- stores a device credential locally
- keeps an outbound WebSocket connection to the relay
- starts Claude Code when the PWA opens a session

## Requirements

Each agent machine needs:

- macOS or Linux
- Claude Code installed
- Claude Code authenticated
- network access to the relay URL

Verify Claude:

```bash
which claude
claude --version
claude --print "Reply with OK"
```

## Build From Source

Until public release binaries exist, build from source:

```bash
go build -o remote-cli ./cmd/agent
chmod +x remote-cli
```

Optional install:

```bash
mkdir -p ~/.local/bin
mv remote-cli ~/.local/bin/
```

Make sure `~/.local/bin` is on your `PATH`.

## Release Binaries

Once tagged releases exist, download the binary for your OS/architecture from GitHub Releases.

Expected release assets:

```text
remote-cli-agent-darwin-arm64
remote-cli-agent-darwin-amd64
remote-cli-agent-linux-arm64
remote-cli-agent-linux-amd64
checksums.txt
```

Example:

```bash
curl -LO <release-url>/remote-cli-agent-darwin-arm64
chmod +x remote-cli-agent-darwin-arm64
mv remote-cli-agent-darwin-arm64 ~/.local/bin/remote-cli
```

## Pair

```bash
remote-cli pair --relay http://YOUR_RELAY_URL
```

The command prints:

- QR code
- 6-digit code
- pairing URL

Use the PWA to scan or enter the code.

Config is written to:

```text
~/.config/remote-cli/config.toml
```

## Run

```bash
remote-cli run
```

Keep it running. The PWA should show the device online.

## Check Status

```bash
remote-cli status
```

This shows whether the machine is paired and which relay URL is stored.

## Unpair

```bash
remote-cli unpair
```

This removes the local config. It does not currently delete the device record from the relay.

## Autostart

There is no built-in service installer yet.

For now, run the agent manually or create your own systemd/launchd service. See [service-autostart.md](service-autostart.md).
