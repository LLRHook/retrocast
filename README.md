# Retrocast

A self-hosted Discord clone inspired by the simplicity of Discord circa 2015-2016. Text channels, voice chat, roles, permissions — no bloat. Built for a small friend group to run on a single machine.

## Architecture

```
retrocast/          Go backend (REST API + WebSocket gateway, admin CLI, compose stack)
retrocast-web/      React web client (Vite, TypeScript)
retrocast-client/   SwiftUI iOS client
docs/               Product brief, SRS, data contracts and architecture
```

**Server stack**: Go, PostgreSQL, Redis, MinIO (file storage), LiveKit (voice), Caddy (TLS
edge), Prometheus and Grafana (monitoring)

## Specification

The requirements and design live in `docs/`:

- [Product brief](docs/PRODUCT_BRIEF.md): goals, scope, release plan and the owner's decisions
- [Software requirements (SRS)](docs/SRS.md) with the
  [server](docs/srs/server.md), [web](docs/srs/web.md), [iOS](docs/srs/ios.md) and
  [system](docs/srs/system.md) requirements
- [Data contracts](docs/DATA_CONTRACTS.md): configuration, entities, REST API, gateway events
- [Architecture](docs/ARCHITECTURE.md): components, deployment, decisions, known defects
- [OpenAPI](retrocast/docs/openapi.yaml): REST contract; current version recorded in [Data contracts](docs/DATA_CONTRACTS.md)

The older prose documents are kept in [docs/legacy](docs/legacy/) for reference; each names
the document that replaces it.

### Server layout

```
cmd/retrocast/         Application entrypoint
cmd/retrocast-cli/     Admin CLI (migrate, seed, health, version)
internal/
  api/                 REST handlers and routing (Echo)
  auth/                Argon2id passwords, JWT tokens, middleware
  config/              Environment-based configuration
  database/            PostgreSQL repositories (pgx)
  gateway/             WebSocket server, presence, typing indicators
  models/              Domain structs
  permissions/         Bitfield RBAC with channel overrides
  redis/               Redis client (refresh tokens, rate limiting, presence)
  service/             Business rules and permission checks
  snowflake/           Distributed ID generation
  storage/             MinIO attachment storage
migrations/            SQL schema migrations (up/down)
```

## Quick start

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

### Run with Docker Compose

```bash
cd retrocast
cp .env.example .env # replace development secrets before exposing the host
docker compose up --build
```

This starts the API server and the web client behind Caddy on ports 80 and 443, along with
PostgreSQL, Redis, MinIO, LiveKit, Prometheus and Grafana. Only Caddy 80/443 and
LiveKit 7880/7881 plus UDP 50000-50100 are public. Grafana binds to
`127.0.0.1:3000`; on a remote host use `ssh -L 3000:127.0.0.1:3000 <host>`.
Set `SERVER_URL=https://<host>` for `retrocast-cli health` and
`RETROCAST_URL=https://<host>` for `scripts/demo.sh`. The demo trusts the exact
local development hosts `localhost` and `127.0.0.1` without certificate verification
and prints a warning; every other host retains TLS verification (use
`CURL_CA_BUNDLE` for a private CA).

MinIO, PostgreSQL, Redis, Prometheus and API 8080 have no host bindings. For
operator access use `docker compose exec api curl http://minio:9000/minio/health/live`
or `docker compose exec db psql -U retrocast`. For the MinIO console, tunnel a
host loopback port to the container's 9001 port using an operator-controlled
forwarder on the compose network; do not publish it on all interfaces.
Attachment downloads use 15-minute signed HTTPS URLs through Caddy (D-10 fixed); multipart uploads remain supported and upload presigning remains Later.
Prometheus scrapes `api:8080/metrics` internally; Caddy returns 404 for `/metrics`.
`https://<host>/docs` serves the embedded raw OpenAPI specification without
external scripts or stylesheets.

### Run locally (development)

Run the compose stack as above. Start the web dev server from `retrocast-web/`
with `npx -y pnpm@10 dev`; its proxy targets `https://localhost` through Caddy
and accepts Caddy's local certificate. Set `RETROCAST_PROXY_URL` to change that
target. Add `http://localhost:5173` to the API's `ALLOWED_ORIGINS` in `.env`
(and recreate the API container) for the Vite browser origin. Extra browser
origins are opt-in; separate entries with commas and include scheme and port.
Use origins without a trailing slash or path.
The instance's own external origin and native clients without Origin are always
served. Direct Go development requires an explicitly configured local dependency
stack; the default compose deliberately keeps dependency ports internal.

### Registration

Registration defaults to `REGISTRATION_MODE=invite`, including upgrades that omit the key. On an empty instance, register the first account without an invite, sign in, create a server, then create invites for friends. Further registrations must send `invite_code` to `/api/v1/auth/register`; account creation, invite consumption and guild membership commit together. Set `open` for unrestricted registration or `closed` to disable registration while preserving sign-in. Unknown mode values stop startup. The web invite-code field is pending WEB-FR-020 (B5); use the API for invited registrations until then. `retrocast/scripts/demo.sh` bootstraps alice and registers bob using her guild invite, and can run again with those existing accounts.

In `open` mode the server ignores `invite_code`; use API-050 after registering to join its guild. `closed` admits nobody even on an empty instance. To prepare a closed instance, bootstrap its first account in `invite` mode, then switch to `closed`.

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://retrocast:password@localhost:5432/retrocast?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `JWT_SECRET` | `change-me-in-production` | Secret for signing JWT tokens |
| `REGISTRATION_MODE` | `invite` | `open`, `invite` or `closed`; unknown values stop startup |
| `ALLOWED_ORIGINS` | empty | Comma-separated additional browser origins, e.g. `http://localhost:5173` |
| `SERVER_ADDR` | `:8080` | HTTP listen address |
| `MINIO_ENDPOINT` | — | MinIO S3-compatible endpoint |
| `MINIO_ACCESS_KEY` | — | MinIO access key |
| `MINIO_SECRET_KEY` | — | MinIO secret key |
| `LIVEKIT_URL` | — | LiveKit server URL |
| `LIVEKIT_API_KEY` | — | LiveKit API key |
| `LIVEKIT_API_SECRET` | — | LiveKit API secret |

### Running tests

```bash
cd retrocast
make test
```

## Contributing

All changes to `main` must go through a pull request with at least one approved review. Read the [specification](#specification) before making changes; [SYSTEM_DESIGN.md](SYSTEM_DESIGN.md) records the original design intent.

## License

This project is unlicensed. All rights reserved.

Attachment downloads use 15-minute signed HTTPS URLs through Caddy. Set `DOMAIN` in `retrocast/.env` to the public hostname (default `localhost`, no scheme, path or port); Compose passes it to both Caddy and the API. Set `MINIO_REGION` consistently (default `us-east-1`). Only the `retrocast` bucket download path is proxied; MinIO remains private.
