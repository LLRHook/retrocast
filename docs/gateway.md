# WebSocket Gateway

The gateway handles real-time event delivery over WebSocket. It runs in-process alongside the REST API in the same Go binary.

Source: `retrocast/internal/gateway/`

## Endpoint

```
GET /gateway
```

No authentication required for the WebSocket upgrade. Authentication happens via IDENTIFY after connection.

## Protocol Overview

The protocol follows a Discord-style flow:

```
Client                          Server
  |                               |
  |  --- WebSocket connect -----> |
  |  <--- Op 10 HELLO ---------- |  (includes heartbeat_interval)
  |  --- Op 2 IDENTIFY --------> |  (access token)
  |  <--- Op 0 READY ----------- |  (session_id, user_id, guild IDs)
  |                               |
  |  <--- Op 1 HEARTBEAT ------- |  (server sends periodically)
  |  --- Op 1 HEARTBEAT -------> |  (client responds)
  |  <--- Op 11 HEARTBEAT_ACK -- |
  |                               |
  |  <--- Op 0 DISPATCH -------- |  (real-time events)
  |  ...                          |
```

## Op Codes

Defined in `retrocast/internal/gateway/events.go`:

| Op | Name | Direction | Description |
|----|------|-----------|-------------|
| 0 | DISPATCH | Server -> Client | Real-time event with `t` (event name) and `s` (sequence) |
| 1 | HEARTBEAT | Both | Server sends to check liveness; client echoes back |
| 2 | IDENTIFY | Client -> Server | Send access token to authenticate |
| 3 | PRESENCE_UPDATE | Client -> Server | Update user's presence status |
| 4 | VOICE_STATE_UPDATE | Server -> Client | Voice channel state changes |
| 6 | RESUME | Client -> Server | Resume a previous session after disconnect |
| 7 | RECONNECT | Server -> Client | Server requests client reconnect |
| 10 | HELLO | Server -> Client | Initial payload with heartbeat interval |
| 11 | HEARTBEAT_ACK | Server -> Client | Acknowledgment of client heartbeat |

Note: Op 5 is intentionally unused (mirrors Discord's protocol gap).

## Payload Format

Every gateway message uses this JSON envelope:

```json
{
    "op": 0,
    "d": { ... },
    "s": 42,
    "t": "MESSAGE_CREATE"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `op` | int | Operation code |
| `d` | object/null | Event data payload |
| `s` | int64/null | Sequence number (only for Op 0 DISPATCH) |
| `t` | string/null | Event name (only for Op 0 DISPATCH) |

Defined as `GatewayPayload` struct in `events.go`.

## Connection Lifecycle

### 1. HELLO (Op 10)

Sent immediately after WebSocket upgrade. See `server.go`:

```json
{"op": 10, "d": {"heartbeat_interval": 41250}}
```

The heartbeat interval is 41,250ms (~41s). Defined as `heartbeatInterval` in `connection.go`.

### 2. IDENTIFY (Op 2)

Client sends access token:

```json
{"op": 2, "d": {"token": "eyJhbGciOiJIUzI1NiIs..."}}
```

Server validates the JWT, generates a session ID (UUID), subscribes the user to all their guilds, sets presence to "online" in Redis, and responds with READY.

### 3. READY (Op 0, Event)

```json
{
    "op": 0,
    "d": {"session_id": "abc-123", "user_id": "12345", "guilds": ["67890"]},
    "s": 1,
    "t": "READY"
}
```

### 4. Heartbeat Loop

The server sends HEARTBEAT (Op 1) every 41.25 seconds. The client must respond with HEARTBEAT (Op 1). Server sends HEARTBEAT_ACK (Op 11) in response.

If no heartbeat is received within `heartbeatInterval + 10s`, the connection is terminated.

See `connection.go:writePump()` and `handleMessage()`.

## Event Types

20 dispatch event types, defined in `events.go`:

### Message Events

| Event | When | Payload |
|-------|------|---------|
| `MESSAGE_CREATE` | Message sent | Full `MessageWithAuthor` |
| `MESSAGE_UPDATE` | Message edited | Updated `MessageWithAuthor` |
| `MESSAGE_DELETE` | Message deleted | `{id, channel_id}` |

### Guild Events

| Event | When | Payload |
|-------|------|---------|
| `GUILD_CREATE` | User joins a guild | Full `Guild` |
| `GUILD_UPDATE` | Guild settings changed | Updated `Guild` |
| `GUILD_DELETE` | Guild deleted or user removed | `{id}` |

### Channel Events

| Event | When | Payload |
|-------|------|---------|
| `CHANNEL_CREATE` | Channel created | Full `Channel` |
| `CHANNEL_UPDATE` | Channel settings changed | Updated `Channel` |
| `CHANNEL_DELETE` | Channel deleted | `{id}` |

### Member Events

| Event | When | Payload |
|-------|------|---------|
| `GUILD_MEMBER_ADD` | User joins guild | `Member` + guild_id |
| `GUILD_MEMBER_REMOVE` | User kicked/left | `{guild_id, user_id}` |
| `GUILD_MEMBER_UPDATE` | Nickname/roles changed | Updated `Member` |

### Role Events

| Event | When | Payload |
|-------|------|---------|
| `GUILD_ROLE_CREATE` | Role created | Full `Role` |
| `GUILD_ROLE_UPDATE` | Role permissions/name changed | Updated `Role` |
| `GUILD_ROLE_DELETE` | Role deleted | `{guild_id, role_id}` |

### Presence & Interaction

| Event | When | Payload |
|-------|------|---------|
| `TYPING_START` | User starts typing | `{channel_id, guild_id, user_id, timestamp}` |
| `PRESENCE_UPDATE` | User status changes | `{user_id, status}` |
| `VOICE_STATE_UPDATE` | Voice channel join/leave | Voice state data |

### Moderation

| Event | When | Payload |
|-------|------|---------|
| `GUILD_BAN_ADD` | User banned | Ban data |
| `GUILD_BAN_REMOVE` | User unbanned | `{guild_id, user_id}` |

## Event Dispatch

The `Manager` (in `manager.go`) maintains three indexes:

```go
connections   map[int64]*Connection            // userID -> connection
subscriptions map[int64]map[int64]bool          // guildID -> set of userIDs
sessions      map[string]*Connection            // sessionID -> connection
```

Three dispatch methods:

| Method | Description |
|--------|-------------|
| `DispatchToGuild(guildID, event, data)` | Send to all users subscribed to the guild |
| `DispatchToUser(userID, event, data)` | Send to a specific user |
| `DispatchToGuildExcept(guildID, exceptUserID, event, data)` | Send to all guild subscribers except one |

Guild events are also stored in the replay buffer for RESUME support.

## Session Resume

### Resume Flow

When a client disconnects and reconnects, it can resume its session to receive missed events:

```
Client                          Server
  |  --- WebSocket connect -----> |
  |  <--- Op 10 HELLO ---------- |
  |  --- Op 6 RESUME ----------> |  {token, session_id, seq}
  |  <--- missed events -------- |  (replayed from ring buffer)
```

### Ring Buffer

Each guild has a ring buffer (`replayBufferSize = 100` events). Events are stored with monotonically increasing sequence numbers. On RESUME, the server replays all events with sequence > client's last known sequence.

Defined in `manager.go`:

```go
replayBuffer map[int64]*ringBuffer  // guildID -> ring buffer
```

If the session is invalid or too many events were missed, the server sends Op 7 RECONNECT, forcing a full IDENTIFY.

## Presence

Presence is tracked in Redis and broadcast to all guilds the user belongs to.

| Status | Description |
|--------|-------------|
| `online` | Connected and active |
| `idle` | User set idle |
| `dnd` | Do Not Disturb |
| `invisible` | Stored as "offline" in Redis/broadcast |
| `offline` | Disconnected |

Client sends Op 3 to update status. On disconnect, presence is cleared after a 10-second grace period (allows reconnection without flashing offline). See `clearPresenceWithGrace()` in `manager.go`.

## Typing Indicators

Handled by `TypingHandler` in `typing.go`. When `POST /channels/:id/typing` is called, the handler:

1. Validates channel exists
2. Stores typing state in Redis (with auto-expiry)
3. Dispatches `TYPING_START` to the guild via `DispatchToGuild()`

## Connection Parameters

Defined in `connection.go`:

| Constant | Value | Purpose |
|----------|-------|---------|
| `heartbeatInterval` | 41,250 ms | Server heartbeat frequency |
| `heartbeatTimeout` | 10 s | Grace period after missed heartbeat |
| `writeWait` | 10 s | Write deadline per message |
| `pongWait` | 60 s | Read deadline (reset on pong) |
| `maxMessageSize` | 4,096 bytes | Max inbound message size |
| `sendBufferSize` | 256 | Outbound channel buffer size |

## Single Connection Per User

If a user connects from a second device, the existing connection receives Op 7 RECONNECT and is closed. Only one active connection per user is supported.

See `register()` in `manager.go`.
