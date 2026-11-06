# Retrocast

A self-hosted Discord clone inspired by the simplicity of Discord circa 2015-2016. Text channels, voice chat, roles, permissions — no bloat. Built for a small friend group to run on a single machine.

## Architecture

```
retrocast/          Go backend (REST API + WebSocket gateway)
retrocast-client/   iOS/macOS client (planned)
```

**Server stack**: Go with a required in-process Rust permission resolver, PostgreSQL, Redis, MinIO (file storage), LiveKit (voice)

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
  permissions/         Go bitfields and cgo bridge to the Rust resolver
  redis/               Redis client (caching, rate limiting, presence)
  snowflake/           Distributed ID generation
native/permissions/    Required Rust base/channel resolver (static C ABI)
migrations/            SQL schema migrations (up/down)
```

## Quick start

### Prerequisites

- [Go 1.25.7+](https://go.dev/dl/)
- For host builds: [Rust 1.93+](https://www.rust-lang.org/tools/install), Cargo, Make, and a C compiler (Linux or macOS, matching the Go host architecture). CI pins Rust 1.99.1. Docker builds include the native toolchain.
- `CGO_ENABLED=1` is required. The Rust archive is linked into the binary; no Rust toolchain or separate Rust library is needed at runtime.
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
make run   # Builds the required Rust archive and Go binary before starting
```

For a direct Go build, run `make native` first, then
`CGO_ENABLED=1 go build -ldflags='-linkmode=external' -o bin/retrocast ./cmd/retrocast`.
Direct `go test` also requires that native build. `CGO_ENABLED=0`, missing native
archives, and cross-compilation without a matching native toolchain are unsupported
and fail the build. `CARGO_TARGET_DIR` may select a shared Cargo cache.
The native build also generates a Go source fingerprint of the archive, so a
Rust-only change invalidates Go's cgo/link cache. Re-run `make native` before
direct Go commands whenever native sources or artifacts change.
The digest is retained in native error diagnostics so it changes the compiled
package identity as well as the source compilation key.
See [permission resolution](docs/permissions.md) for the ABI and error policy.

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
make test              # Existing full race suite; database tests need test services
make native-test       # Rust unit/ABI checks, Clippy, formatting (no services)
make cache-check        # Focused stable-path native relink regression (no services)
make permission-test   # Real Go/Rust oracle parity and repository fakes (no services)
make package-check     # Relocated binary, missing archive, CGO/ABI faults (no services)
```

## Contributing

All changes to `main` must go through a pull request with at least one approved review. See the [system design document](SYSTEM_DESIGN.md) for architecture details before making changes.

## License

This project is unlicensed. All rights reserved.
