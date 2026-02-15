# API Reference

All endpoints are defined in `retrocast/internal/api/router.go`. Base path: `/api/v1`.

## Response Format

**Success**: `{"data": <payload>}` (status 200/201)

**Error**: `{"error": {"code": "ERROR_CODE", "message": "Human-readable message"}}` (status 4xx/5xx)

**No content**: Empty body (status 204) for DELETE operations and typing indicators.

## Authentication

Protected endpoints require a Bearer token in the Authorization header:

```
Authorization: Bearer <access_token>
```

Access tokens are 15-minute HS256 JWTs. When expired, use the refresh endpoint to get a new pair.

## Rate Limits

- **Auth routes** (`/api/v1/auth/*`): 5 requests/minute per IP
- **Protected routes**: 50 requests/minute per user
- Rate limit exceeded: `429 {"error": {"code": "RATE_LIMITED", "message": "..."}}`

---

## Auth (4 endpoints)

### POST /api/v1/auth/register

Create a new user account. No auth required.

```json
// Request
{"username": "victor", "password": "s3cret", "display_name": "Victor"}

// Response 201
{"data": {"access_token": "eyJ...", "refresh_token": "a1b2c3..."}}
```

### POST /api/v1/auth/login

Authenticate with credentials. No auth required.

```json
// Request
{"username": "victor", "password": "s3cret"}

// Response 200
{"data": {"access_token": "eyJ...", "refresh_token": "a1b2c3..."}}
```

### POST /api/v1/auth/refresh

Exchange a refresh token for new token pair. No auth required.

```json
// Request
{"refresh_token": "a1b2c3..."}

// Response 200
{"data": {"access_token": "eyJ...", "refresh_token": "d4e5f6..."}}
```

### POST /api/v1/auth/logout

Invalidate the current refresh token. Requires auth.

```
// Response 204 (no content)
```

---

## Users (3 endpoints)

### GET /api/v1/users/@me

Get the current authenticated user's profile.

```json
// Response 200
{"data": {"id": "12345", "username": "victor", "display_name": "Victor", "avatar_hash": null, "created_at": "2025-01-15T..."}}
```

### PATCH /api/v1/users/@me

Update the current user's profile.

```json
// Request
{"display_name": "Victor I.", "avatar_hash": "abc123"}

// Response 200
{"data": { ... updated user ... }}
```

### GET /api/v1/users/@me/guilds

List all guilds the current user is a member of.

```json
// Response 200
{"data": [{"id": "67890", "name": "Friends", "owner_id": "12345", ...}]}
```

---

## DM Channels (2 endpoints)

### POST /api/v1/users/@me/channels

Create or get a DM channel with another user.

```json
// Request
{"recipient_id": "99999"}

// Response 200
{"data": {"id": "11111", "type": 1, "recipients": [...], "created_at": "..."}}
```

### GET /api/v1/users/@me/channels

List the current user's DM channels.

```json
// Response 200
{"data": [{"id": "11111", "type": 1, "recipients": [...]}]}
```

---

## Guilds (4 endpoints)

### POST /api/v1/guilds

Create a new guild. The creator becomes the owner.

```json
// Request
{"name": "My Server"}

// Response 201
{"data": {"id": "67890", "name": "My Server", "owner_id": "12345", ...}}
```

### GET /api/v1/guilds/:id

Get guild details.

### PATCH /api/v1/guilds/:id

Update guild settings. Requires `MANAGE_GUILD` permission.

```json
// Request
{"name": "New Name"}
```

### DELETE /api/v1/guilds/:id

Delete a guild. Owner only.

---

## Channels (5 endpoints)

### POST /api/v1/guilds/:id/channels

Create a channel. Requires `MANAGE_CHANNELS` permission.

```json
// Request
{"name": "general", "type": 0, "parent_id": "55555"}
```

Channel types: `0` = text, `2` = voice, `4` = category.

### GET /api/v1/guilds/:id/channels

List all channels in a guild.

### GET /api/v1/channels/:id

Get a single channel.

### PATCH /api/v1/channels/:id

Update channel name/topic. Requires `MANAGE_CHANNELS`.

```json
{"name": "renamed", "topic": "New topic"}
```

### DELETE /api/v1/channels/:id

Delete a channel. Requires `MANAGE_CHANNELS`.

---

## Members (6 endpoints)

### GET /api/v1/guilds/:id/members

List guild members. Supports pagination via query params.

Query: `?limit=100&offset=0`

### GET /api/v1/guilds/:id/members/:user_id

Get a specific member.

### PATCH /api/v1/guilds/:id/members/:user_id

Update a member's nickname. Requires `MANAGE_NICKNAMES`.

```json
{"nickname": "Cool Nick"}
```

### PATCH /api/v1/guilds/:id/members/@me

Update your own nickname. Requires `CHANGE_NICKNAME`.

```json
{"nickname": "My Nick"}
```

### DELETE /api/v1/guilds/:id/members/:user_id

Kick a member. Requires `KICK_MEMBERS`.

### DELETE /api/v1/guilds/:id/members/@me

Leave a guild. Always allowed (except for the owner).

---

## Roles (6 endpoints)

### POST /api/v1/guilds/:id/roles

Create a role. Requires `MANAGE_ROLES`.

```json
{"name": "Moderator", "permissions": 100, "color": 3447003}
```

### GET /api/v1/guilds/:id/roles

List all roles in a guild.

### PATCH /api/v1/guilds/:id/roles/:role_id

Update a role. Requires `MANAGE_ROLES`.

```json
{"name": "Admin", "permissions": 2147483648, "color": 15158332}
```

### DELETE /api/v1/guilds/:id/roles/:role_id

Delete a role. Requires `MANAGE_ROLES`.

### PUT /api/v1/guilds/:id/members/:user_id/roles/:role_id

Assign a role to a member. Requires `MANAGE_ROLES`.

### DELETE /api/v1/guilds/:id/members/:user_id/roles/:role_id

Remove a role from a member. Requires `MANAGE_ROLES`.

---

## Channel Permission Overrides (2 endpoints)

### PUT /api/v1/channels/:id/permissions/:role_id

Set permission overrides for a role in a channel. Requires `MANAGE_ROLES`.

```json
{"allow": 3, "deny": 4}
```

### DELETE /api/v1/channels/:id/permissions/:role_id

Remove permission overrides for a role in a channel. Requires `MANAGE_ROLES`.

---

## Messages (5 endpoints)

### POST /api/v1/channels/:id/messages

Send a message. Requires `SEND_MESSAGES` in the channel.

```json
// Request
{"content": "Hello world!"}

// Response 201
{"data": {"id": "99999", "channel_id": "44444", "author_id": "12345", "content": "Hello world!", "created_at": "...", "author_username": "victor", "author_display_name": "Victor", "attachments": []}}
```

### GET /api/v1/channels/:id/messages

Get message history with cursor-based pagination. Requires `READ_MESSAGE_HISTORY`.

Query: `?limit=50&before=99998` (before = Snowflake ID cursor)

Returns messages newest-first.

### GET /api/v1/channels/:id/messages/:message_id

Get a single message.

### PATCH /api/v1/channels/:id/messages/:message_id

Edit a message. Only the author can edit.

```json
{"content": "Updated content"}
```

### DELETE /api/v1/channels/:id/messages/:message_id

Delete a message. Author can delete own; `MANAGE_MESSAGES` required for others'.

---

## Attachments (1 endpoint)

### POST /api/v1/channels/:id/attachments

Upload a file as multipart form data. Requires `ATTACH_FILES` in the channel.

```
Content-Type: multipart/form-data; boundary=...

--boundary
Content-Disposition: form-data; name="file"; filename="image.png"
Content-Type: image/png

<binary data>
--boundary--
```

```json
// Response 201
{"id": "88888", "message_id": "0", "filename": "image.png", "content_type": "image/png", "size": 12345, "url": "https://..."}
```

---

## Typing (1 endpoint)

### POST /api/v1/channels/:id/typing

Trigger a typing indicator. Dispatches `TYPING_START` event to the guild.

```
// Response 204 (no content)
```

---

## Bans (3 endpoints)

### PUT /api/v1/guilds/:id/bans/:user_id

Ban a member. Requires `BAN_MEMBERS`.

```json
{"reason": "Spamming"}
```

### DELETE /api/v1/guilds/:id/bans/:user_id

Unban a user. Requires `BAN_MEMBERS`.

### GET /api/v1/guilds/:id/bans

List all bans in a guild. Requires `BAN_MEMBERS`.

---

## Invites (4 endpoints + 1 public)

### GET /api/v1/invites/:code (Public -- no auth)

Get invite info (guild name, member count).

### POST /api/v1/guilds/:id/invites

Create an invite. Requires `CREATE_INVITE`.

```json
{"max_uses": 10, "max_age_seconds": 86400}
```

### GET /api/v1/guilds/:id/invites

List all invites for a guild.

### POST /api/v1/invites/:code

Accept an invite and join the guild.

### DELETE /api/v1/invites/:code

Revoke an invite.

---

## Infrastructure Endpoints

### GET /health

Deep health check. Pings Postgres and Redis.

```json
// 200
{"status": "ok"}

// 503
{"status": "error", "component": "postgres"}
```

### GET /metrics

Prometheus metrics endpoint (echoprometheus middleware).

### GET /gateway

WebSocket upgrade endpoint. See [gateway.md](gateway.md).

---

## Endpoint Summary (46 total)

| Category | Count | Auth Required |
|----------|-------|---------------|
| Auth | 4 | 1 of 4 |
| Users | 3 | Yes |
| DMs | 2 | Yes |
| Guilds | 4 | Yes |
| Channels | 5 | Yes |
| Members | 6 | Yes |
| Roles | 6 | Yes |
| Channel Overrides | 2 | Yes |
| Messages | 5 | Yes |
| Attachments | 1 | Yes |
| Typing | 1 | Yes |
| Bans | 3 | Yes |
| Invites | 5 | 4 of 5 |
| Infrastructure | 3 | No |
| **Total** | **50** | |
