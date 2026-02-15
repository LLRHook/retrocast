# Implementation Plan — P0-P2 Full Build

## Execution Phases

Six phases. Each phase runs parallel tracks. Phases execute sequentially (each depends on prior).

---

## Phase A: Backend Features (4 parallel tracks)

All backend features are independent. Run simultaneously.

### Track A1: File Uploads (MinIO)

Attachments table already exists (`migrations/000008`). Need model, repo, handler, MinIO client.

| Step | File | Action |
|------|------|--------|
| 1 | `internal/models/attachment.go` | Create — `Attachment` struct (ID, MessageID, Filename, ContentType, Size, StorageKey, URL) |
| 2 | `internal/database/repositories.go` | Edit — add `AttachmentRepository` interface (Create, GetByMessageID, Delete) |
| 3 | `internal/database/attachment_repo.go` | Create — Postgres implementation |
| 4 | `internal/storage/minio.go` | Create — MinIO client wrapper (Upload, GetURL, Delete) using `github.com/minio/minio-go/v7` |
| 5 | `internal/api/upload_handler.go` | Create — `POST /channels/:id/attachments` accepts multipart, validates type/size (10MB max), uploads to MinIO, stores metadata |
| 6 | `internal/models/message.go` | Edit — add `Attachments []Attachment` field to `MessageWithAuthor` |
| 7 | `internal/database/message_repo.go` | Edit — join attachments in GetByID/GetByChannelID queries |
| 8 | `internal/api/router.go` | Edit — wire upload route |
| 9 | `cmd/retrocast/main.go` | Edit — create MinIO client, attachment repo, upload handler |
| 10 | `internal/api/upload_handler_test.go` | Create — tests for upload validation, size limits |

### Track A2: Ban System

| Step | File | Action |
|------|------|--------|
| 1 | `migrations/000013_create_bans.up.sql` | Create — `bans` table (guild_id, user_id, reason, created_by, created_at, PK on guild_id+user_id) |
| 2 | `migrations/000013_create_bans.down.sql` | Create — `DROP TABLE bans` |
| 3 | `internal/models/ban.go` | Create — `Ban` struct |
| 4 | `internal/database/repositories.go` | Edit — add `BanRepository` interface (Create, GetByGuildAndUser, GetByGuildID, Delete) |
| 5 | `internal/database/ban_repo.go` | Create — Postgres implementation |
| 6 | `internal/api/ban_handler.go` | Create — PUT (ban), DELETE (unban), GET (list bans). Ban auto-kicks member. Uses `PermBanMembers`. |
| 7 | `internal/api/invite_handler.go` | Edit — check ban list in `AcceptInvite` before creating member |
| 8 | `internal/api/router.go` | Edit — wire ban routes under `/guilds/:id/bans` |
| 9 | `cmd/retrocast/main.go` | Edit — create ban repo, ban handler, add to deps |
| 10 | `internal/api/ban_handler_test.go` | Create — tests for ban/unban/list + invite rejection |
| 11 | `gateway/events.go` | Edit — add `EventGuildBanAdd`, `EventGuildBanRemove` constants (if not already defined) |

### Track A3: DMs / Private Channels

| Step | File | Action |
|------|------|--------|
| 1 | `migrations/000014_create_dm_channels.up.sql` | Create — `dm_channels` (id, type, created_at) + `dm_recipients` (channel_id, user_id) |
| 2 | `migrations/000014_create_dm_channels.down.sql` | Create — drop tables |
| 3 | `internal/models/dm_channel.go` | Create — `DMChannel` struct with recipients |
| 4 | `internal/database/repositories.go` | Edit — add `DMChannelRepository` interface (Create, GetByID, GetByUserID, GetOrCreateDM, AddRecipient) |
| 5 | `internal/database/dm_repo.go` | Create — Postgres implementation |
| 6 | `internal/api/dm_handler.go` | Create — `POST /users/@me/channels` (create/get DM), `GET /users/@me/channels` (list DMs) |
| 7 | `internal/api/message_handler.go` | Edit — support DM channels (skip guild permission checks, check recipient membership instead) |
| 8 | `internal/api/router.go` | Edit — wire DM routes |
| 9 | `cmd/retrocast/main.go` | Edit — create DM repo, DM handler |
| 10 | `internal/api/dm_handler_test.go` | Create — tests |

### Track A4: Missing Backend Tests

| Step | File | Action |
|------|------|--------|
| 1 | `internal/api/user_handler_test.go` | Create — TestGetMe (auth user returned), TestGetMe_Unauthorized, TestUpdateMe (display name, avatar), TestUpdateMe_EmptyPayload |
| 2 | `internal/api/ratelimit_test.go` | Create — TestRateLimit_Allowed, TestRateLimit_Exceeded, TestRateLimit_FailOpen (redis error) |
| 3 | `internal/api/testutil_test.go` | Edit — add `mockBanRepo` and `mockDMChannelRepo` for new features |

**Checkpoint A:** `make test` passes with all new + existing tests. Build succeeds.

---

## Phase B: Deployment Hardening (1 track, all independent tasks)

| Step | File | Action |
|------|------|--------|
| 1 | `.github/workflows/ci.yml` | Create — lint + test + build on push/PR, Postgres + Redis services |
| 2 | `go.mod` | Edit — add `github.com/golang-migrate/migrate/v4` + postgres driver |
| 3 | `cmd/retrocast/main.go` | Edit — run migrations after pool creation, before handler setup |
| 4 | `docker-compose.yml` | Edit — Redis: add `command: redis-server --appendonly yes` + volume. Add Caddy service. |
| 5 | `Caddyfile` | Create — reverse proxy to api:8080 |
| 6 | `cmd/retrocast/main.go` | Edit — replace `log.Printf/Fatalf` with `slog.Info/Error` |
| 7 | `go.mod` | Edit — add `github.com/labstack/echo-contrib/echoprometheus` |
| 8 | `internal/api/router.go` | Edit — add `GET /metrics` endpoint |
| 9 | `scripts/backup.sh` | Create — pg_dump wrapper |
| 10 | `.env.example` | Create — document all env vars |

**Checkpoint B:** `docker compose build` succeeds. CI config is valid YAML. `make build` succeeds.

---

## Phase C: Frontend Foundation (2 parallel tracks)

**Prerequisite:** Install XcodeGen (`brew install xcodegen`).

### Track C1: Project Scaffold + Models + Networking

| Step | File | Action |
|------|------|--------|
| 1 | `retrocast-client/Retrocast/project.yml` | Create — XcodeGen spec for iOS 17+ SwiftUI app |
| 2 | `retrocast-client/Retrocast/Sources/RetrocastApp.swift` | Create — @main entry, environment objects |
| 3 | `retrocast-client/Retrocast/Sources/Models/Snowflake.swift` | Create — Int64 wrapper with string JSON coding |
| 4 | `retrocast-client/Retrocast/Sources/Models/User.swift` | Create — Codable struct |
| 5 | `retrocast-client/Retrocast/Sources/Models/Guild.swift` | Create — Codable struct |
| 6 | `retrocast-client/Retrocast/Sources/Models/Channel.swift` | Create — Codable struct with ChannelType enum |
| 7 | `retrocast-client/Retrocast/Sources/Models/Message.swift` | Create — Codable struct with author fields |
| 8 | `retrocast-client/Retrocast/Sources/Models/Member.swift` | Create — Codable struct |
| 9 | `retrocast-client/Retrocast/Sources/Models/Role.swift` | Create — Codable struct |
| 10 | `retrocast-client/Retrocast/Sources/Models/Invite.swift` | Create — Codable struct |
| 11 | `retrocast-client/Retrocast/Sources/Networking/APIError.swift` | Create — error types matching server codes |
| 12 | `retrocast-client/Retrocast/Sources/Networking/Endpoints.swift` | Create — all ~30 endpoint definitions |
| 13 | `retrocast-client/Retrocast/Sources/Networking/APIClient.swift` | Create — URLSession wrapper, auth injection, token refresh |
| 14 | `retrocast-client/Retrocast/Sources/Networking/TokenManager.swift` | Create — Keychain storage, auto-refresh |
| 15 | `retrocast-client/Retrocast/Sources/Utilities/KeychainHelper.swift` | Create — Security framework wrapper |
| 16 | `retrocast-client/Retrocast/Sources/Utilities/DateFormatters.swift` | Create — relative timestamps |

### Track C2: Gateway + State + ViewModels

| Step | File | Action |
|------|------|--------|
| 1 | `retrocast-client/Retrocast/Sources/Gateway/GatewayPayload.swift` | Create — op codes, payload encoding |
| 2 | `retrocast-client/Retrocast/Sources/Gateway/GatewayEvent.swift` | Create — event type enum |
| 3 | `retrocast-client/Retrocast/Sources/Gateway/ReconnectionStrategy.swift` | Create — exponential backoff with jitter |
| 4 | `retrocast-client/Retrocast/Sources/Gateway/GatewayClient.swift` | Create — WebSocket connection, heartbeat, identify, resume, reconnect |
| 5 | `retrocast-client/Retrocast/Sources/State/AppState.swift` | Create — @Observable, guilds, channels, messages, selection |
| 6 | `retrocast-client/Retrocast/Sources/State/PresenceState.swift` | Create — user presence map |
| 7 | `retrocast-client/Retrocast/Sources/ViewModels/AuthViewModel.swift` | Create — login/register flow |
| 8 | `retrocast-client/Retrocast/Sources/ViewModels/ServerListViewModel.swift` | Create — guild list, create/join |
| 9 | `retrocast-client/Retrocast/Sources/ViewModels/ChannelListViewModel.swift` | Create — channel sidebar for selected guild |
| 10 | `retrocast-client/Retrocast/Sources/ViewModels/ChatViewModel.swift` | Create — message list, send, load history, cursor pagination |

**Checkpoint C:** `xcodegen generate` succeeds. Project opens in Xcode. `swift build` compiles (may need adjustments for iOS-only APIs).

---

## Phase D: Frontend Views — Phase 1 (2 parallel tracks)

### Track D1: Auth + App Structure

| Step | File | Action |
|------|------|--------|
| 1 | `Sources/Views/ContentView.swift` | Create — auth gate (logged in → MainView, else → AuthFlowView) |
| 2 | `Sources/Views/Auth/ServerAddressView.swift` | Create — server URL entry + health check |
| 3 | `Sources/Views/Auth/LoginView.swift` | Create — username/password form |
| 4 | `Sources/Views/Auth/RegisterView.swift` | Create — registration form |
| 5 | `Sources/Components/AvatarView.swift` | Create — circle image with initials fallback |
| 6 | `Sources/Components/LoadingView.swift` | Create — spinner |
| 7 | `Sources/Components/ErrorBanner.swift` | Create — toast-style error |
| 8 | `Sources/Components/GuildIcon.swift` | Create — server icon with morph |
| 9 | `Sources/Utilities/Colors.swift` | Create — color palette (retroDark, retroAccent, etc.) |

### Track D2: Main Navigation + Chat

| Step | File | Action |
|------|------|--------|
| 1 | `Sources/Views/Main/MainView.swift` | Create — NavigationSplitView (3-column) |
| 2 | `Sources/Views/Main/ServerListView.swift` | Create — guild icon strip |
| 3 | `Sources/Views/Main/ChannelSidebarView.swift` | Create — channel list grouped by category |
| 4 | `Sources/Views/Main/ChatAreaView.swift` | Create — messages + input wrapper |
| 5 | `Sources/Views/Chat/MessageListView.swift` | Create — ScrollView with lazy loading |
| 6 | `Sources/Views/Chat/MessageRow.swift` | Create — avatar + name + content + timestamp |
| 7 | `Sources/Views/Chat/MessageInput.swift` | Create — text field + send button |

**Checkpoint D:** App compiles and runs in Simulator. User can: enter server address → login/register → see guilds → see channels → send/receive messages.

---

## Phase E: Frontend Phase 2 — Rich Chat (2 parallel tracks)

### Track E1: Message Enhancements

| Step | File | Action |
|------|------|--------|
| 1 | `Sources/Views/Chat/MessageRow.swift` | Edit — grouped messages (same author within 5 min) |
| 2 | `Sources/Views/Chat/DateSeparator.swift` | Create — "January 15, 2025" between days |
| 3 | `Sources/Views/Chat/MessageListView.swift` | Edit — infinite scroll (load older on scroll to top) |
| 4 | `Sources/Views/Chat/MessageRow.swift` | Edit — long-press context menu for edit/delete |
| 5 | `Sources/Utilities/MarkdownParser.swift` | Create — basic markdown → AttributedString |

### Track E2: Members + Presence + Typing

| Step | File | Action |
|------|------|--------|
| 1 | `Sources/ViewModels/MemberListViewModel.swift` | Create — fetch members, group by role |
| 2 | `Sources/Views/Members/MemberListView.swift` | Create — right sidebar |
| 3 | `Sources/Views/Members/MemberRow.swift` | Create — avatar + name + presence |
| 4 | `Sources/Components/PresenceDot.swift` | Create — green/yellow/red/gray dot |
| 5 | `Sources/Components/RoleTag.swift` | Create — colored role pill |
| 6 | `Sources/Views/Chat/TypingIndicator.swift` | Create — "User is typing..." bar |
| 7 | `Sources/Views/Members/UserProfilePopover.swift` | Create — tap on member → profile card |

**Checkpoint E:** Message grouping works. Typing indicators show. Member list displays with presence. Infinite scroll loads history.

---

## Phase F: Frontend Phase 3 — Guild Management (2 parallel tracks)

### Track F1: Invites + Guild Creation

| Step | File | Action |
|------|------|--------|
| 1 | `Sources/ViewModels/InviteViewModel.swift` | Create — create/accept invites |
| 2 | `Sources/Views/Guild/CreateGuildSheet.swift` | Create — name input + create |
| 3 | `Sources/Views/Guild/JoinGuildSheet.swift` | Create — invite code input |
| 4 | `Sources/Views/Guild/InviteSheet.swift` | Create — generate link + share sheet |
| 5 | `Sources/Views/Guild/GuildSettingsView.swift` | Create — edit name/icon, manage roles, manage channels |

### Track F2: Settings + Admin Actions

| Step | File | Action |
|------|------|--------|
| 1 | `Sources/ViewModels/SettingsViewModel.swift` | Create — user settings, server settings |
| 2 | `Sources/Views/Settings/UserSettingsView.swift` | Create — display name, avatar |
| 3 | `Sources/Views/Settings/AppSettingsView.swift` | Create — theme, notifications placeholder |
| 4 | Edit `ChannelSidebarView.swift` | Add — create/edit/delete channel buttons (admin only) |
| 5 | Edit `MemberRow.swift` | Add — long-press → kick/ban options |

**Checkpoint F:** Full guild management. Create/join guilds. Invite system. Role display. Channel management. User settings.

---

## Verification

After all phases:
1. `cd retrocast && make test` — all tests pass with `-race`
2. `cd retrocast && make build` — binary builds
3. `cd retrocast && docker compose build` — container builds
4. Xcode project opens and compiles for iOS Simulator
5. Manual smoke test: register → create guild → send message → verify real-time delivery

---

## File Count Summary

| Area | New Files | Modified Files |
|------|-----------|----------------|
| Backend | ~15 | ~8 |
| Deployment | ~5 | ~3 |
| Frontend | ~50 | ~5 (later phases edit earlier files) |
| **Total** | **~70** | **~16** |
