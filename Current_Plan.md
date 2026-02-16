# Execution Plan — Full "Up Next" TODO

## Batch 1 — Foundation (Parallel)

### 1A: Web Client Phase 1 — Foundation
1. Scaffold `retrocast-web/` with Vite 6 + React 19 + TypeScript + Tailwind CSS v4 + React Router v7 (pnpm)
2. API client module (`src/lib/api.ts`): fetch wrapper, Bearer auth, 401 auto-refresh with request coalescing, response envelope unwrap
3. Auth Zustand store (`src/stores/auth.ts`): user, tokens, login/register/logout actions, localStorage persistence for refresh token
4. Auth pages (`src/pages/`): ServerAddress → Login/Register → redirect to app
5. Gateway client (`src/lib/gateway.ts`): native WebSocket, Discord opcodes (HELLO→IDENTIFY→READY), heartbeat loop, RESUME with session_id + seq, exponential backoff reconnect (1s base, 60s max, 10 attempts)

### 1B: Backend — Message Search
1. Migration `015_add_message_search.up.sql`: Add `tsvector` column + GIN index to messages, trigger for auto-update
2. `MessageSearchRepo` in `internal/database/message_search_repo.go`
3. `SearchService` in `internal/service/search.go`
4. Handler: `GET /api/v1/guilds/:id/messages/search?q=&author_id=&before=&after=&limit=`
5. Handler test
6. Update OpenAPI spec

### 1C: Backend — Read States
1. Migration `016_create_read_states.up.sql`: `read_states(user_id, channel_id, last_message_id, mention_count, updated_at)`
2. `ReadStateRepo` in `internal/database/read_state_repo.go`
3. `ReadStateService` in `internal/service/read_state.go`
4. Handler: `PUT /api/v1/channels/:id/ack/:message_id` (mark as read)
5. Include unread info in READY event payload
6. Handler test + repo test
7. Update OpenAPI spec

### 1D: Backend — Message Reactions
1. Migration `017_create_reactions.up.sql`: `reactions(message_id, user_id, emoji, created_at)` with unique constraint
2. `ReactionRepo` in `internal/database/reaction_repo.go`
3. `ReactionService` in `internal/service/reaction.go`
4. Handlers: `PUT /channels/:id/messages/:mid/reactions/:emoji/@me`, `DELETE .../reactions/:emoji/@me`, `GET .../reactions/:emoji`
5. Gateway events: `MESSAGE_REACTION_ADD`, `MESSAGE_REACTION_REMOVE`
6. Handler tests
7. Update OpenAPI spec

### 1E: Docs — Update CLAUDE.md
1. Update CLAUDE.md "Backend Architecture" section to reflect service layer (handlers → services → repos)
2. Update dependency injection description
3. Add service layer to repository structure tree

---

## Batch 2 — Layout & More Backend (Parallel)

### 2A: Web Client Phase 2 — Core Layout & Navigation
1. App shell layout (`src/layouts/AppLayout.tsx`): 3-column Discord layout (server list 72px, channel sidebar 240px, main area flex)
2. Zustand stores (`src/stores/`): guilds, channels, members, roles, messages, presence, typing, DMs
3. Gateway event dispatcher (`src/lib/gateway-dispatcher.ts`): route DISPATCH events to Zustand stores
4. Guild list sidebar (`src/components/ServerList.tsx`): guild icons, selection indicator, create/join modals
5. Channel sidebar (`src/components/ChannelSidebar.tsx`): channels grouped by category, text/voice icons, auto-select first text channel

### 2B: Backend — Voice/LiveKit Integration
1. Add `livekit-server-sdk-go` dependency
2. Migration `018_create_voice_states.up.sql`: `voice_states(guild_id, channel_id, user_id, session_id, self_mute, self_deaf)`
3. `VoiceRepo` in `internal/database/voice_repo.go`
4. `VoiceService` in `internal/service/voice.go`: join/leave room, generate LiveKit tokens, track state
5. Handle `OpVoiceStateUpdate` in gateway server
6. Dispatch `EventVoiceStateUpdate` events
7. Handler: `POST /api/v1/channels/:id/voice/join` (returns LiveKit token)
8. Handler tests
9. Update OpenAPI spec

### 2C: Backend — Group DMs
1. Migration `019_create_group_dm_members.up.sql`: extend DM system for groups (dm_channel_members table, owner_id on dm_channels)
2. Extend `DMRepo` for group operations
3. Extend `DMService`: create group DM, add/remove members, rename
4. Handlers: `POST /users/@me/channels` (extended for group), `PUT/DELETE /channels/:id/recipients/:user_id`
5. Handler tests
6. Update OpenAPI spec

---

## Batch 3 — Messaging

### 3A: Web Client Phase 3 — Messaging
1. Message list view (`src/components/MessageList.tsx`): cursor-based pagination, date separators, 5-min author grouping, scroll-to-bottom
2. Message input (`src/components/MessageInput.tsx`): multiline, Enter to send, Shift+Enter newline, typing throttle 8s
3. Message context menu: edit own, delete own, copy text
4. File upload: multipart POST, image preview, drag-and-drop
5. Markdown rendering: react-markdown + remark-gfm, code blocks, links, inline formatting
6. Typing indicator display: animated dots, "X is typing..." / "Several people are typing..."

---

## Batch 4 — Members & iOS Search (Parallel)

### 4A: Web Client Phase 4 — Members & Presence
1. Member list panel (right sidebar, toggleable, grouped by highest role)
2. User profile popover (avatar, display name, presence, roles, join date)
3. Presence dots (online green, idle yellow, dnd red, offline gray)
4. Avatar component (image URL with initials fallback, lazy loading)

### 4B: iOS — Message Search UI
1. Search bar in channel view
2. Search results list with message previews
3. Navigate to message on tap
4. Calls `GET /api/v1/guilds/:id/messages/search`

---

## Batch 5 — DMs, Invites & Settings

### 5A: Web Client Phase 5 — DMs, Invites & Settings
1. DM list view (list open DMs, create new DM, switch between guild/DM mode)
2. DM conversations (reuse message list/input, DM-specific header)
3. Invite system (generate code, copy to clipboard, view/revoke invites, accept via code/URL)
4. Guild settings (edit name, delete/leave, role list)
5. Role editor (name, color picker, 19 permission toggles in 4 groups, create/edit/delete)
6. User settings (edit display name, logout)
7. Channel creation/edit (create modal with name/type/category, context menu rename/delete)

---

## Batch 6 — Polish & iOS Voice (Parallel)

### 6A: Web Client Phase 6 — Polish & CI
1. Dark theme (Discord palette: #313338 main, #2b2d31 sidebar, #1e1f22 server list)
2. Responsive/adaptive layout (collapsible sidebars for smaller viewports)
3. CI job for web client (pnpm install, lint, typecheck, build)
4. Dockerfile for web client (nginx serving static build)

### 6B: iOS — Voice Channel UI
1. Voice channel join/leave buttons
2. Voice state indicators (who's in channel, mute/deaf icons)
3. Self-mute/deafen controls
4. Calls `POST /api/v1/channels/:id/voice/join` for LiveKit token

---

## Verification Checkpoints

After each batch:
- Backend: `go test -v -race ./...` + `golangci-lint run ./...` + `go build ./cmd/retrocast`
- Web: `pnpm lint && pnpm tsc --noEmit && pnpm build` (from Batch 1 onward)
- iOS: Xcode build check (for iOS batches)
- Commit after each batch passes
