# Retrocast

A self-hosted Discord clone inspired by the simplicity of Discord circa 2015-2016. Text channels, voice chat, roles, permissions — no bloat. Built for a small friend group to run on a single machine.

## Architecture

```
retrocast/          Go backend (REST API + WebSocket gateway)
retrocast-client/   iOS/macOS client (planned)
```

**Server stack**: Go, PostgreSQL, Redis, MinIO (file storage), LiveKit (voice)

### Server layout

```
cmd/retrocast/         Application entrypoint
internal/
  api/                 REST handlers and routing (Echo)
  auth/                Argon2id passwords, JWT tokens, middleware
  config/              Environment-based configuration
  database/            PostgreSQL repositories (pgx)
  gateway/             WebSocket server, presence, typing indicators
  models/              Domain structs
  permissions/         Bitfield RBAC with channel overrides
  redis/               Redis client (caching, rate limiting, presence)
  snowflake/           Distributed ID generation
migrations/            SQL schema migrations (up/down)
```

## Quick start

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

### Run with Docker Compose

```bash
cd retrocast
docker compose up --build
```

This starts the API server on `:8080` along with PostgreSQL, Redis, MinIO, and LiveKit.

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

All changes to `main` must go through a pull request with at least one approved review. See the [system design document](SYSTEM_DESIGN.md) for architecture details before making changes.

## License

This project is unlicensed. All rights reserved.
