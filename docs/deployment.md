# Deployment

Retrocast is deployed as a set of Docker containers orchestrated by Docker Compose. This document covers the container setup, reverse proxy, monitoring, environment configuration, and backup procedures.

## Docker Compose Stack

Defined in `retrocast/docker-compose.yml`. Seven services:

| Service | Image | Purpose | Ports |
|---------|-------|---------|-------|
| `api` | Built from `retrocast/Dockerfile` | Go server (REST + WebSocket) | 8080 |
| `db` | `postgres:17` | Primary database | internal |
| `redis` | `redis:7-alpine` | Cache, sessions, presence, rate limits | internal |
| `minio` | `minio/minio:latest` | S3-compatible file storage | 9000 (API), 9001 (console) |
| `livekit` | `livekit/livekit-server:latest` | WebRTC voice/video SFU | 7880, 7881, 50000-50100/udp |
| `caddy` | `caddy:2-alpine` | Reverse proxy with auto-TLS | 80, 443 |
| `prometheus` | `prom/prometheus:latest` | Metrics collection | 9090 |

## Dockerfile

Multi-stage build in `retrocast/Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o retrocast ./cmd/retrocast

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl && adduser -D -u 1001 appuser
COPY --from=builder /app/retrocast /usr/local/bin/retrocast
COPY --from=builder /app/migrations /migrations
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1
CMD ["retrocast"]
```

Key details:
- Runs as non-root user `appuser` (UID 1001)
- Migrations are copied into the image at `/migrations`
- Built-in Docker health check hitting `GET /health`
- Final image is Alpine 3.21 (~5 MB base)

## Service Configuration

### API Service

```yaml
api:
  build: .
  ports: ["8080:8080"]
  env_file: .env
  environment:
    - DATABASE_URL=postgres://retrocast:${POSTGRES_PASSWORD}@db:5432/retrocast?sslmode=disable
    - REDIS_URL=redis://redis:6379
    - MINIO_ENDPOINT=minio:9000
    - LIVEKIT_URL=ws://livekit:7880
  depends_on:
    db: { condition: service_healthy }
    redis: { condition: service_started }
  restart: unless-stopped
```

The API waits for Postgres to be healthy before starting. Migrations auto-run on startup.

### PostgreSQL

```yaml
db:
  image: postgres:17
  volumes: [pgdata:/var/lib/postgresql/data]
  env_file: .env
  environment: [POSTGRES_DB=retrocast, POSTGRES_USER=retrocast]
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U retrocast"]
    interval: 5s
    timeout: 5s
    retries: 5
```

Data persisted in `pgdata` Docker volume. Password from `POSTGRES_PASSWORD` in `.env`.

### Redis

```yaml
redis:
  image: redis:7-alpine
  command: redis-server --appendonly yes
  volumes: [redis-data:/data]
```

AOF persistence enabled. Data is ephemeral (sessions, presence, rate limits) but persistence prevents unnecessary reconnections on restarts.

### MinIO

```yaml
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  ports: ["9000:9000", "9001:9001"]
  volumes: [minio-data:/data]
  env_file: .env
  environment: [MINIO_ROOT_USER=retrocast]
```

Web console available at port 9001. Password from `MINIO_ROOT_PASSWORD` in `.env`.

### LiveKit

```yaml
livekit:
  image: livekit/livekit-server:latest
  ports: ["7880:7880", "7881:7881", "50000-50100:50000-50100/udp"]
  env_file: .env
  environment: [LIVEKIT_KEYS=${LIVEKIT_API_KEY}:${LIVEKIT_API_SECRET}]
```

UDP ports 50000-50100 for WebRTC media streams.

## Caddy (Reverse Proxy)

Configuration in `retrocast/Caddyfile`:

```
{$DOMAIN:localhost} {
    reverse_proxy api:8080
}
```

- Automatic HTTPS with Let's Encrypt when `DOMAIN` is set to a public hostname
- Falls back to `localhost` (HTTP) when `DOMAIN` is not set
- WebSocket connections are proxied transparently
- Data stored in `caddy_data` volume

## Prometheus (Monitoring)

Configuration in `retrocast/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: retrocast
    static_configs:
      - targets: ['api:8080']
```

Scrapes the `GET /metrics` endpoint every 15 seconds. The API uses the `echoprometheus` middleware to expose request counts, latencies, and other HTTP metrics.

Access Prometheus UI at `http://localhost:9090`.

## Environment Variables

All secrets and configuration are managed via `.env` file (referenced by `env_file: .env` in docker-compose). Use `.env.example` as a template.

### Required

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | HMAC secret for signing JWTs |
| `POSTGRES_PASSWORD` | Database password (used by both PG and API) |

### Optional (with defaults)

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `SERVER_ADDR` | `:8080` | API listen address |
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

### Service-Specific (no defaults)

| Variable | Description |
|----------|-------------|
| `MINIO_ENDPOINT` | MinIO server address (e.g., `minio:9000`) |
| `MINIO_ACCESS_KEY` | MinIO access key |
| `MINIO_SECRET_KEY` | MinIO secret key |
| `MINIO_ROOT_PASSWORD` | MinIO root password (for the container) |
| `LIVEKIT_URL` | LiveKit WebSocket URL |
| `LIVEKIT_API_KEY` | LiveKit API key |
| `LIVEKIT_API_SECRET` | LiveKit API secret |
| `DOMAIN` | Public hostname for Caddy (optional, defaults to `localhost`) |

### Config File Support

The API also supports a config file as an alternative to environment variables. Search order:

1. `$RETROCAST_CONFIG` path
2. `./retrocast.toml` (working directory)
3. `/etc/retrocast/config.toml`

Format: `KEY = VALUE` (one per line, `#` comments). Environment variables always override file values.

See `retrocast/internal/config/config.go`.

## Startup

```bash
cd retrocast
cp .env.example .env    # Edit with your secrets
docker compose up -d
```

The API will:
1. Connect to PostgreSQL (waits for health check)
2. Run all pending migrations automatically
3. Connect to Redis
4. Initialize MinIO client
5. Start the HTTP server and WebSocket gateway

## Health Check

```
GET /health
```

Returns `200 {"status": "ok"}` when both Postgres and Redis are reachable. Returns `503` with the failing component name otherwise. Used by Docker's built-in health check.

## Volumes

| Volume | Service | Data |
|--------|---------|------|
| `pgdata` | db | PostgreSQL data files |
| `redis-data` | redis | Redis AOF persistence |
| `minio-data` | minio | Uploaded files and attachments |
| `caddy_data` | caddy | TLS certificates |
| `prometheus-data` | prometheus | Metrics time series |

## Backup & Restore

### PostgreSQL

```bash
# Backup
docker compose exec db pg_dump -U retrocast retrocast > backup.sql

# Restore
docker compose exec -T db psql -U retrocast retrocast < backup.sql
```

### MinIO

```bash
# Backup (copy volume)
docker compose stop minio
docker run --rm -v retrocast_minio-data:/data -v $(pwd):/backup alpine \
    tar czf /backup/minio-backup.tar.gz /data
docker compose start minio
```

### Full Stack

```bash
# Stop, backup all volumes, restart
docker compose down
for vol in pgdata redis-data minio-data; do
    docker run --rm -v "retrocast_${vol}:/data" -v $(pwd)/backups:/backup alpine \
        tar czf "/backup/${vol}.tar.gz" /data
done
docker compose up -d
```

## Local Development

For local development without Docker:

```bash
cd retrocast
make build          # Build binary to bin/retrocast
make test           # Run all tests (race detector enabled)
make run            # Build and run
```

Requires a running PostgreSQL and Redis instance. Set `DATABASE_URL` and `REDIS_URL` environment variables or use a `retrocast.toml` config file.
