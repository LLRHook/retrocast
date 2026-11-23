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
- [OpenAPI](retrocast/docs/openapi.yaml): REST contract v2.0.0

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
docker compose up --build
```

This starts the API server and the web client behind Caddy on ports 80 and 443, along with
PostgreSQL, Redis, MinIO, LiveKit, Prometheus and Grafana. The compose file also publishes the
API on `:8080` today; the deployment target publishes only Caddy and LiveKit
([ARCHITECTURE §6](docs/ARCHITECTURE.md#6-deployment-view)).

### Run locally (development)

Start the dependencies:

```bash
cd retrocast
docker compose up db redis minio livekit
```

Then build and run the server:

```bash
make run
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://retrocast:password@localhost:5432/retrocast?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `JWT_SECRET` | `change-me-in-production` | Secret for signing JWT tokens |
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
