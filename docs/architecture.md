# Architecture

Retrocast is a self-hosted Discord-like chat platform. This document describes the system architecture, component boundaries, and how the layers fit together.

## System Overview

The platform consists of two main codebases:

- **Backend** (`retrocast/`) -- Go monolithic server handling REST API, WebSocket gateway, and all business logic
- **iOS Client** (`retrocast-client/`) -- Native SwiftUI app with MVVM architecture

Supporting infrastructure runs via Docker Compose: PostgreSQL, Redis, MinIO, LiveKit, Caddy, and Prometheus.

## Component Diagram

```
                    iOS / Web / Desktop
                         |
                    HTTPS / WSS
                         |
                   +-----------+
                   |   Caddy   |  (reverse proxy, auto-TLS)
                   +-----------+
                         |
              +---------------------+
              |  Go API + Gateway   |  retrocast/cmd/retrocast/main.go
              |   (Echo v4.15)      |
              +---------------------+
              /    |       |       \
         +----+  +-----+ +-----+ +-------+
         | PG |  |Redis| |MinIO| |LiveKit|
         +----+  +-----+ +-----+ +-------+
```

## Layers

The backend follows a three-layer architecture. There is no separate service layer -- handlers own business logic directly.

### 1. HTTP Handlers (Business Logic)

Location: `retrocast/internal/api/`

Each handler owns the business logic for its resource domain:

| Handler | File | Responsibility |
|---------|------|---------------|
| `AuthHandler` | `auth_handler.go` | Registration, login, token refresh, logout |
| `GuildHandler` | `guild_handler.go` | CRUD for guilds, permission middleware factory |
| `ChannelHandler` | `channel_handler.go` | CRUD for channels within guilds |
| `MessageHandler` | `message_handler.go` | Send, edit, delete messages; cursor pagination |
| `MemberHandler` | `member_handler.go` | List, update, kick members; leave guild |
| `RoleHandler` | `role_handler.go` | CRUD for roles, assign/remove, channel overrides |
| `InviteHandler` | `invite_handler.go` | Create, accept, revoke invite links |
| `BanHandler` | `ban_handler.go` | Ban, unban, list bans |
| `DMHandler` | `dm_handler.go` | Create and list DM channels |
| `UploadHandler` | `upload_handler.go` | File upload via multipart form |
| `UserHandler` | `user_handler.go` | Get/update current user profile |

### 2. Repositories (Data Access)

Location: `retrocast/internal/database/`

Repository interfaces are defined in `repositories.go`. Implementations live in separate `*_repo.go` files. All methods accept `context.Context` as first argument.

Repositories: `UserRepository`, `GuildRepository`, `ChannelRepository`, `RoleRepository`, `MemberRepository`, `MessageRepository`, `InviteRepository`, `ChannelOverrideRepository`, `AttachmentRepository`, `BanRepository`, `DMChannelRepository`.

### 3. Gateway (Real-Time Events)

Location: `retrocast/internal/gateway/`

The gateway runs in-process alongside the REST API. It maintains WebSocket connections and pushes real-time events to clients. See [gateway.md](gateway.md) for the full protocol.

## Dependency Injection

Manual DI in `cmd/retrocast/main.go`. Initialization order:

1. **Infrastructure**: Postgres pool, Redis client, Snowflake generator, JWT TokenService
2. **Repositories**: All receive the same `pgxpool.Pool`
3. **Storage**: MinIO client
4. **Gateway**: `Manager` receives TokenService, GuildRepository, Redis
5. **Handlers**: Each receives its specific repos + shared services
6. **Dependencies struct**: Aggregates everything, passed to `api.SetupRouter()`

```go
deps := &api.Dependencies{
    Auth:         authHandler,
    Guilds:       guildHandler,
    Channels:     channelHandler,
    // ... all handlers ...
    Gateway:      gwManager,
    TokenService: tokenSvc,
    Pool:         pool,
    Redis:        rdb,
}
```

## Request/Response Pattern

All API responses use a consistent envelope:

- **Success**: `{"data": <payload>}` via `successJSON(c, status, data)`
- **Error**: `{"error": {"code": "ERROR_CODE", "message": "..."}}` via `Error(c, status, code, message)`

Defined in `retrocast/internal/api/response.go`.

## Authentication Flow

1. JWT middleware extracts Bearer token from Authorization header
2. Token validated via `TokenService.ValidateAccessToken()`
3. `user_id` set in Echo context
4. Handlers retrieve it with `auth.GetUserID(c)`

Tokens: 15-min HS256 JWTs for access, opaque hex strings (32 bytes) stored in Redis for refresh (7-day TTL).

See `retrocast/internal/auth/jwt.go`.

## Rate Limiting

Defined in `retrocast/internal/api/ratelimit.go`.

- **Unauthenticated**: per-IP, keyed as `rl:ip:{ip}:{path}`
- **Authenticated**: per-user, keyed as `rl:user:{uid}:{path}`
- Auth routes: 5 requests/minute
- Protected routes: 50 requests/minute
- Fail-open on Redis errors (requests allowed through)

## IDs

All entities use Snowflake IDs (64-bit integers, JSON-marshaled as strings).

Defined in `retrocast/internal/snowflake/snowflake.go`:

```
Bits: [42 timestamp] [5 worker] [5 process] [12 sequence]
Epoch: Jan 1, 2025 00:00:00 UTC (1735689600000 ms)
```

IDs are chronologically sortable and globally unique. The generator is initialized with worker=1, process=1 in `main.go`.

## Configuration

Three-tier hierarchy (defined in `retrocast/internal/config/config.go`):

1. Environment variables (highest priority)
2. Config file (`retrocast.toml`, `/etc/retrocast/config.toml`, or `$RETROCAST_CONFIG`)
3. Hardcoded defaults (lowest priority)

Required: `DATABASE_URL`, `JWT_SECRET`. Optional with defaults: `REDIS_URL` (`redis://localhost:6379`), `SERVER_ADDR` (`:8080`).

## Observability

- **Health check**: `GET /health` -- pings Postgres and Redis
- **Metrics**: `GET /metrics` -- Prometheus via `echoprometheus` middleware
- **Logging**: Structured JSON via `slog` (configurable level: `LOG_LEVEL`)

## Key Dependencies

| Dependency | Purpose | Version |
|-----------|---------|---------|
| Echo | HTTP framework | v4.15 |
| pgx | PostgreSQL driver | v5 |
| go-redis | Redis client | v9 |
| gorilla/websocket | WebSocket connections | latest |
| golang-jwt | JWT tokens | v5 |
| golang-migrate | Database migrations | v4 |
| minio-go | S3-compatible storage | v7 |
| echoprometheus | Metrics middleware | latest |
