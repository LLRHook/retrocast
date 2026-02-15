# TODO

## Done

### Backend
- [x] Auth system (Argon2id passwords, JWT access tokens, refresh tokens in Redis)
- [x] Guilds CRUD
- [x] Channels CRUD
- [x] Messages with cursor-based pagination
- [x] Members management
- [x] Roles with hierarchy
- [x] Permissions (guild-level + channel overrides, bitfield RBAC)
- [x] Invites
- [x] WebSocket gateway (HELLO/IDENTIFY/READY/heartbeat/RESUME)
- [x] Typing indicators
- [x] Presence tracking
- [x] Rate limiting (per-IP unauth, per-user auth)
- [x] File uploads (MinIO/S3)
- [x] DMs
- [x] Ban system
- [x] All handler tests

### Infra
- [x] Health check endpoint
- [x] CI with linting (golangci-lint)
- [x] Prometheus metrics
- [x] Backup automation
- [x] Docker Compose full stack (API + Postgres + Redis + MinIO + LiveKit + Caddy)

## In Progress

### Backend
- [ ] Extract service layer between handlers and repos
- [ ] Fix permission constant duplication (use `permissions` package everywhere)
- [ ] Add repository-layer tests (target: happy-path + error-path per method)
- [ ] Add OpenAPI/Swagger docs (swaggo/swag or manual openapi.yaml)
- [ ] Build CLI tool (`retrocast-cli` for scripting, bots, automation)
- [ ] Config file support (~/.retrocast.conf fallback)
- [ ] Standardized CLI exit codes
- [ ] Demo bootstrap script (scripts/demo.sh)

### Docs
- [ ] Split SYSTEM_DESIGN.md into focused docs/ files
- [ ] Add verification section to CLAUDE.md
- [ ] Add commit message convention to CLAUDE.md
- [ ] Keep project structure tree in CLAUDE.md current

## Up Next

### Backend
- [ ] Voice/video via LiveKit integration
- [ ] Message search (full-text)
- [ ] Read states / unread badges

### iOS
- [ ] DM UI (conversation list + chat view)
- [ ] Avatar image loading
- [ ] Markdown rendering in messages
- [ ] Channel creation UI
