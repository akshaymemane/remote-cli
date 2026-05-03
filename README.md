# remote-cli

> **Alpha**: self-hosted, early, and intended for technical users comfortable running a relay and an agent process.

remote-cli is a self-hosted mobile control plane for Claude Code sessions across your machines.

Your laptop, desktop, and Raspberry Pi show up like devices in a chat app. Tap one from your phone, start a Claude Code session on that machine, send prompts, and watch responses stream back through the relay.

```text
Phone / PWA  <-->  Relay  <-->  Agent on each machine  -->  claude CLI
```

The relay is the hub. Agents and phones connect outbound to the relay, so your agent machines do not need inbound ports.

## Status

This project is not production-ready yet. It is suitable for private alpha testing and dogfooding.

Current alpha limitations:

- The agent runs in the foreground with `remote-cli run`, or as a background service via `remote-cli service install`.
- Claude Code must already be installed and authenticated on every agent machine.
- Tool-use cards are currently observational; phone approval/deny is not a supported permission path yet.
- The relay sees plaintext messages. There is no end-to-end encryption in v1.
- Chat history is not persisted across reloads/restarts yet.

## Components

- **Relay**: Go HTTP/WebSocket server. Handles login, pairing, device presence, session routing, SQLite state, and serves the PWA.
- **Agent**: Go CLI installed on each controlled machine. Pairs with the relay, keeps an outbound WebSocket connection, and spawns `claude` for sessions.
- **PWA**: React/Vite mobile app served by the relay. Provides login, device list, pairing, and chat UI.

## Quickstart

For the detailed version, see [docs/quickstart.md](docs/quickstart.md).

### 1. Start the relay

Clone the repo:

```bash
git clone https://github.com/akshaymemane/remote-cli.git
cd remote-cli
cp .env.example .env
```

Edit `.env`:

```bash
RELAY_URL=http://YOUR_LAN_IP:8080
RELAY_JWT_SECRET=<generate-with-openssl-rand-hex-32>
RELAY_ADMIN_PASSWORD=<choose-a-password>
```

Start the relay:

```bash
docker compose up -d
```

Open `http://YOUR_LAN_IP:8080` on your phone.

Important: `localhost` usually does not work for multi-device setups. Use a LAN IP, Tailscale hostname, or public HTTPS domain that your phone and every agent machine can reach. See [docs/choosing-relay-url.md](docs/choosing-relay-url.md).

### 2. Install the agent

Download the latest binary for your platform from [Releases](https://github.com/akshaymemane/remote-cli/releases/latest):

```bash
# macOS Apple Silicon
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-darwin-arm64
chmod +x remote-cli && sudo mv remote-cli /usr/local/bin/

# macOS Intel
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-darwin-amd64
chmod +x remote-cli && sudo mv remote-cli /usr/local/bin/

# Linux amd64
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-linux-amd64
chmod +x remote-cli && sudo mv remote-cli /usr/local/bin/

# Linux arm64 (Raspberry Pi etc.)
curl -Lo remote-cli https://github.com/akshaymemane/remote-cli/releases/latest/download/remote-cli-agent-linux-arm64
chmod +x remote-cli && sudo mv remote-cli /usr/local/bin/
```

Or build from source (requires Go 1.23+):

```bash
go build -o remote-cli ./cmd/agent
```

See [docs/agent-install.md](docs/agent-install.md).

### 3. Pair each machine

On every machine you want to control:

```bash
remote-cli pair --relay http://YOUR_RELAY_URL
```

Scan the QR code from the PWA or enter the 6-digit pairing code manually.

### 4. Run the agent

Pairing saves credentials; it does not keep the agent online.

After pairing, run:

```bash
remote-cli run
```

Keep that process running. The device should appear online in the PWA.

## Requirements

- Relay host: Docker and Docker Compose, or Go if running from source.
- Agent machines: Claude Code installed and authenticated; Go 1.23+ only if building from source.
- Phone: modern mobile browser. HTTPS is recommended and may be required for camera QR scanning depending on browser/device.

## Documentation

- [Quickstart](docs/quickstart.md)
- [Choosing Your Relay URL](docs/choosing-relay-url.md)
- [Relay Deployment](docs/relay-deployment.md)
- [Agent Install](docs/agent-install.md)
- [Service And Autostart](docs/service-autostart.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Architecture](docs/architecture.md)
- [Protocol](docs/protocol.md)
- [Development](docs/development.md)
- [Roadmap](docs/roadmap.md)
- [Security](SECURITY.md)
- [Contributing](CONTRIBUTING.md)

## Development

Relay:

```bash
go run ./cmd/relay
```

Agent:

```bash
go run ./cmd/agent pair --relay http://localhost:8080
go run ./cmd/agent run
```

PWA dev server:

```bash
cd pwa
npm install
VITE_RELAY_URL=http://localhost:8080 npm run dev
```

Full stack with relay serving built PWA:

```bash
cd pwa
npm run build
cd ..
go run ./cmd/relay
```

See [docs/development.md](docs/development.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

remote-cli is self-hosted. The relay sees plaintext prompts, assistant responses, tool events, device names, and presence state. Use TLS and run the relay only on infrastructure you trust.

See [SECURITY.md](SECURITY.md) for the full trust model.

## License

MIT. See [LICENSE](LICENSE).
