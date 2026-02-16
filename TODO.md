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
- [x] Rate limiting (per-IP unauth, per-user auth, response headers)
- [x] File uploads (MinIO/S3)
- [x] DMs
- [x] Ban system
- [x] All handler tests
- [x] Service layer extraction (handlers → services → repos)
- [x] Repository-layer tests (12 files, 68 tests)
- [x] OpenAPI docs (docs/openapi.yaml)
- [x] CLI tool (retrocast-cli)
- [x] Standardized CLI exit codes
- [x] Config file support (~/.retrocast.conf fallback)
- [x] Demo bootstrap script (scripts/demo.sh)
- [x] Voice/video via LiveKit integration
- [x] Message search (full-text with tsvector + GIN index)
- [x] Read states / unread badges
- [x] Group DMs
- [x] Message reactions/emojis

### Infra
- [x] Health check endpoint
- [x] CI with linting (golangci-lint v2)
- [x] CI Docker build + push to GHCR
- [x] Prometheus metrics
- [x] Grafana dashboards (provisioned)
- [x] Backup automation
- [x] Docker Compose full stack (API + Postgres + Redis + MinIO + LiveKit + Caddy + Prometheus + Grafana)
- [x] Reverse proxy with auto-TLS (Caddy)
- [x] Graceful shutdown
- [x] Structured JSON logging

### Web Client
- [x] Scaffold project (Vite 6 + React 19 + TypeScript + Tailwind CSS v4 + React Router v7, pnpm)
- [x] API client (fetch wrapper, Bearer auth header, 401 → auto token refresh, response envelope unwrap)
- [x] Auth stores + token persistence (Zustand store, localStorage for refresh token)
- [x] Auth pages (server address → login/register → redirect to app)
- [x] Gateway client (native WebSocket, Discord opcodes, heartbeat loop, RESUME, exponential backoff reconnect)
- [x] App shell layout (3-column Discord layout: server list 72px, channel sidebar 240px, main area flex)
- [x] Zustand stores (guilds, channels, members, roles, messages, presence, typing, DMs, users)
- [x] Gateway event dispatcher (route DISPATCH events to Zustand stores)
- [x] Guild list sidebar (guild icons with selection indicator, create/join guild modals)
- [x] Channel sidebar (channels grouped by category, text/voice icons, auto-select first text channel)
- [x] Message list view (cursor-based pagination, date separators, 5-min author grouping, scroll to bottom)
- [x] Message input (multiline, Enter to send, Shift+Enter newline, typing indicator throttle 8s)
- [x] Message context menu (edit own, delete own, copy text)
- [x] File upload (multipart POST, image preview, drag-and-drop)
- [x] Markdown rendering (react-markdown + remark-gfm, code blocks, links, inline formatting)
- [x] Typing indicator display (animated dots, grammar-aware)
- [x] Member list panel (right sidebar, toggleable, members grouped by highest role)
- [x] User profile popover (avatar, display name, presence, roles, join date)
- [x] Presence dots (online green, idle yellow, dnd red, offline gray)
- [x] Avatar component (image URL with initials fallback, userId-hashed color, lazy loading)
- [x] DM list view (list open DMs, create new DM, switch between guild channels and DMs)
- [x] DM conversations (reuse message list/input, DM-specific header)
- [x] Invite system (generate invite code, copy to clipboard, view/revoke invites, accept via code)
- [x] Guild settings (edit name, delete/leave, role list)
- [x] Role editor (name, color picker, 20 permission toggles in 4 groups, create/edit/delete)
- [x] User settings (edit display name, logout)
- [x] Channel creation/edit (create modal with name/type/category, context menu rename/delete)
- [x] Dark theme (Discord palette: #313338 main, #2b2d31 sidebar, #1e1f22 server list)
- [x] Responsive/adaptive layout (collapsible sidebars, hamburger menu for mobile)
- [x] CI job for web client (pnpm install, lint, typecheck, build)
- [x] Dockerfile for web client (nginx serving static build)
- [x] ESLint configuration

### iOS
- [x] Auth flow (register, login, token management)
- [x] Guild list sidebar
- [x] Channel list sidebar
- [x] Message view with cursor pagination
- [x] Real-time message delivery via gateway
- [x] WebSocket client (HELLO/IDENTIFY/READY, heartbeat, RESUME)
- [x] Member list panel
- [x] Typing indicators
- [x] Presence indicators
- [x] Settings (profile, guild, app)
- [x] Invite system UI
- [x] Role/permission management UI
- [x] Responsive layout (NavigationSplitView)
- [x] DM UI (conversation list + chat view)
- [x] Avatar image loading (AsyncImage + initials fallback)
- [x] Markdown rendering in messages
- [x] Channel creation/edit UI
- [x] Message search UI
- [x] Voice channel UI

### Docs
- [x] Commit message convention in CLAUDE.md
- [x] Project structure tree in CLAUDE.md
- [x] Update CLAUDE.md to reflect service layer architecture
