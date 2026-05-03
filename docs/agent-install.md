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

## Release Binaries

Download the latest binary for your OS/architecture from GitHub Releases:

```bash
# macOS Apple Silicon
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-darwin-arm64
chmod +x remote-cli

# macOS Intel
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-darwin-amd64
chmod +x remote-cli

# Linux amd64
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-linux-amd64
chmod +x remote-cli

# Linux arm64
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-linux-arm64
chmod +x remote-cli
```

Install it somewhere on your `PATH`:

```bash
mkdir -p ~/.local/bin
mv remote-cli ~/.local/bin/
```

Make sure `~/.local/bin` is on your `PATH`.

Expected release asset names:

```text
remote-cli-agent-darwin-arm64
remote-cli-agent-darwin-amd64
remote-cli-agent-linux-arm64
remote-cli-agent-linux-amd64
checksums.txt
```

## Build From Source

Requires Go 1.23+:

```bash
go build -o remote-cli ./cmd/agent
chmod +x remote-cli
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

You can also pair and immediately start the agent:

```bash
remote-cli pair --relay http://YOUR_RELAY_URL --run
```

## Check Status

```bash
remote-cli status
```

This shows whether the machine is paired and which relay URL is stored.

## Unpair

```bash
remote-cli unpair
```

This removes the local config. It does not delete the relay-side device record. Delete the device from the PWA if you also want to remove it from the relay.

## Autostart

On macOS and Linux, install the agent as a background service:

```bash
remote-cli service install
remote-cli service logs
```

See [service-autostart.md](service-autostart.md).
