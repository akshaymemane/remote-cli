# Relay Deployment

The relay is the central server. Phones and agents connect to it over HTTP/WebSocket.

Docker Compose is the recommended deployment path for self-hosted beta users.

## Responsibilities

The relay:

- serves the PWA
- authenticates phone sessions
- handles pairing
- stores device/admin state in SQLite
- tracks active WebSocket connections
- routes session messages between phones and agents

## Docker Compose

Create `.env` from the example:

```bash
cp .env.example .env
```

Set:

```bash
RELAY_URL=http://192.168.1.10:8080
RELAY_JWT_SECRET=<generate-with-openssl-rand-hex-32>
RELAY_ADMIN_PASSWORD=<choose-a-password>
```

Start:

```bash
docker compose up -d
```

Check:

```bash
docker compose ps
docker compose logs relay
```

Stop:

```bash
docker compose down
```

## Environment Variables

| Variable | Purpose |
| --- | --- |
| `RELAY_ADDR` | Listen address. Default: `:8080`. |
| `RELAY_DB` | SQLite database path. |
| `RELAY_JWT_SECRET` | Secret for signing phone JWTs. Must be changed. |
| `RELAY_URL` | Public/reachable base URL used for pairing and PWA access. |
| `RELAY_STATIC_DIR` | Directory for built PWA assets. |
| `RELAY_ADMIN_PASSWORD` | Optional non-interactive first-run admin password. |

## SQLite Data

The relay stores:

- admin password hash
- device IDs
- device token hashes
- device names
- last-seen timestamps

With Docker Compose, data is stored in a Docker volume.

Back it up if you care about preserving paired devices.

## TLS

Use TLS for public deployments.

Caddy example:

```caddy
relay.example.com {
    reverse_proxy localhost:8080
}
```

nginx example:

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

## Deployment Options

### LAN

Good for first tests.

```bash
RELAY_URL=http://192.168.1.10:8080
```

### Tailscale

Good for private remote access.

```bash
RELAY_URL=http://relay-host.tailnet-name.ts.net:8080
```

### Public VPS

Good for always-on access.

```bash
RELAY_URL=https://relay.example.com
```

## Running From Source

Build the PWA:

```bash
cd pwa
npm install
npm run build
cd ..
```

Run relay:

```bash
RELAY_URL=http://localhost:8080 go run ./cmd/relay
```

On first run, it prompts for an admin password unless `RELAY_ADMIN_PASSWORD` is set.

## Production Notes

Before public exposure:

- use HTTPS
- set a strong `RELAY_JWT_SECRET`
- use a strong admin password
- protect the SQLite volume
- avoid running on a laptop that sleeps
- prefer Tailscale or a private network while the project is still early
