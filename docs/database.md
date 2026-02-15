# Database

PostgreSQL 17 is the sole persistent data store. All schema changes are managed by 14 migration pairs in `retrocast/migrations/`, auto-applied on startup via `golang-migrate`.

## Connection

Connection pooling via `pgxpool.Pool` (pgx/v5). Pool created in `cmd/retrocast/main.go`:

```go
pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
```

Connection string: `DATABASE_URL` environment variable (required). Format: `postgres://user:pass@host:5432/dbname?sslmode=disable`

## Schema Overview

14 tables, created in dependency order:

```
users
  |
  +-- guilds (owner_id -> users)
  |     |
  |     +-- channels (guild_id -> guilds, parent_id -> channels)
  |     |     |
  |     |     +-- messages (channel_id -> channels, author_id -> users)
  |     |     |     |
  |     |     |     +-- attachments (message_id -> messages)
  |     |     |
  |     |     +-- channel_overrides (channel_id -> channels, role_id -> roles)
  |     |     |
  |     |     +-- invites (guild_id -> guilds, channel_id -> channels)
  |     |
  |     +-- roles (guild_id -> guilds)
  |     |
  |     +-- members (guild_id -> guilds, user_id -> users)
  |     |     |
  |     |     +-- member_roles (guild_id+user_id -> members, role_id -> roles)
  |     |
  |     +-- bans (guild_id -> guilds, user_id -> users)
  |
  +-- refresh_tokens (user_id -> users)
  +-- device_tokens (user_id -> users)
  +-- dm_channels / dm_recipients (user_id -> users)
```

## Table Details

### users (Migration 000001)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| username | VARCHAR(32) | NOT NULL, UNIQUE |
| display_name | VARCHAR(32) | NOT NULL |
| avatar_hash | VARCHAR(64) | nullable |
| password_hash | TEXT | NOT NULL |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

Index: `idx_users_username ON users(username)`

### guilds (Migration 000002)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| name | VARCHAR(100) | NOT NULL |
| icon_hash | VARCHAR(64) | nullable |
| owner_id | BIGINT | FK -> users(id) |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

### channels (Migration 000003)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| guild_id | BIGINT | FK -> guilds ON DELETE CASCADE |
| name | VARCHAR(100) | NOT NULL |
| type | SMALLINT | DEFAULT 0 (0=text, 2=voice, 4=category) |
| position | INT | DEFAULT 0 |
| topic | VARCHAR(1024) | nullable |
| parent_id | BIGINT | FK -> channels (self-ref for categories) |

Unique constraint: `(guild_id, name)`

### roles (Migration 000004)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| guild_id | BIGINT | FK -> guilds ON DELETE CASCADE |
| name | VARCHAR(100) | NOT NULL |
| color | INT | DEFAULT 0 |
| permissions | BIGINT | DEFAULT 0 (bitfield) |
| position | INT | DEFAULT 0 |
| is_default | BOOLEAN | DEFAULT FALSE |

### members (Migration 000005)

| Column | Type | Constraints |
|--------|------|------------|
| guild_id | BIGINT | FK -> guilds ON DELETE CASCADE |
| user_id | BIGINT | FK -> users ON DELETE CASCADE |
| nickname | VARCHAR(32) | nullable |
| joined_at | TIMESTAMPTZ | DEFAULT NOW() |

PK: `(guild_id, user_id)`

### member_roles (Migration 000006)

| Column | Type | Constraints |
|--------|------|------------|
| guild_id | BIGINT | composite FK -> members |
| user_id | BIGINT | composite FK -> members |
| role_id | BIGINT | FK -> roles ON DELETE CASCADE |

PK: `(guild_id, user_id, role_id)`. FK: `(guild_id, user_id)` -> members ON DELETE CASCADE.

### messages (Migration 000007)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| channel_id | BIGINT | FK -> channels ON DELETE CASCADE |
| author_id | BIGINT | FK -> users |
| content | TEXT | NOT NULL |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |
| edited_at | TIMESTAMPTZ | nullable |

Index: `idx_messages_channel_id ON messages(channel_id, id DESC)` -- optimized for cursor-based pagination.

### attachments (Migration 000008)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| message_id | BIGINT | FK -> messages ON DELETE CASCADE |
| filename | VARCHAR(256) | NOT NULL |
| content_type | VARCHAR(128) | NOT NULL |
| size | BIGINT | NOT NULL |
| storage_key | TEXT | NOT NULL |

### invites (Migration 000009)

| Column | Type | Constraints |
|--------|------|------------|
| code | VARCHAR(8) | PK |
| guild_id | BIGINT | FK -> guilds ON DELETE CASCADE |
| channel_id | BIGINT | FK -> channels, nullable |
| creator_id | BIGINT | FK -> users |
| max_uses | INT | DEFAULT 0 (0 = unlimited) |
| uses | INT | DEFAULT 0 |
| expires_at | TIMESTAMPTZ | nullable |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

### channel_overrides (Migration 000010)

| Column | Type | Constraints |
|--------|------|------------|
| channel_id | BIGINT | FK -> channels ON DELETE CASCADE |
| role_id | BIGINT | FK -> roles ON DELETE CASCADE |
| allow_perms | BIGINT | DEFAULT 0 |
| deny_perms | BIGINT | DEFAULT 0 |

PK: `(channel_id, role_id)`. See [permissions.md](permissions.md) for the override algorithm.

### device_tokens (Migration 000011)

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGSERIAL | PK (auto-increment) |
| user_id | BIGINT | FK -> users ON DELETE CASCADE |
| platform | VARCHAR(16) | NOT NULL |
| token | TEXT | NOT NULL |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

Unique: `(user_id, token)`

### refresh_tokens (Migration 000012)

| Column | Type | Constraints |
|--------|------|------------|
| token | VARCHAR(64) | PK (hex-encoded) |
| user_id | BIGINT | FK -> users ON DELETE CASCADE |
| expires_at | TIMESTAMPTZ | NOT NULL |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

### bans (Migration 000013)

| Column | Type | Constraints |
|--------|------|------------|
| guild_id | BIGINT | FK -> guilds ON DELETE CASCADE |
| user_id | BIGINT | FK -> users ON DELETE CASCADE |
| reason | TEXT | nullable |
| created_by | BIGINT | FK -> users |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

PK: `(guild_id, user_id)`

### dm_channels / dm_recipients (Migration 000014)

**dm_channels:**

| Column | Type | Constraints |
|--------|------|------------|
| id | BIGINT | PK (Snowflake) |
| type | INT | DEFAULT 1 (1=DM, 3=group DM) |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

**dm_recipients:**

| Column | Type | Constraints |
|--------|------|------------|
| channel_id | BIGINT | FK -> dm_channels ON DELETE CASCADE |
| user_id | BIGINT | FK -> users ON DELETE CASCADE |

PK: `(channel_id, user_id)`. Index: `idx_dm_recipients_user ON dm_recipients(user_id)`.

## Query Patterns

### Message Pagination (Cursor-Based)

Messages are fetched using cursor-based pagination with the Snowflake ID as cursor:

```sql
SELECT ... FROM messages
WHERE channel_id = $1 AND id < $2
ORDER BY id DESC
LIMIT $3
```

The `idx_messages_channel_id` index on `(channel_id, id DESC)` makes this an index-only scan.

### Repository Interfaces

Defined in `retrocast/internal/database/repositories.go`. Key patterns:

- All reads/writes take `context.Context` for timeout propagation
- Create methods accept pointer to model, set generated fields on it
- Get methods return `(*Model, error)` -- nil model means not found
- List methods return `([]Model, error)`
- `MemberRepository.AddRole/RemoveRole` manage the `member_roles` join table
- `DMChannelRepository.GetOrCreateDM` is an upsert for 1-on-1 DM channels

## Cascade Behavior

All guild-owned resources cascade on guild delete. User deletion cascades members, device tokens, and refresh tokens. Channel deletion cascades messages, overrides, and invites. Message deletion cascades attachments.

## Go Model Conventions

- Snowflake IDs use `json:"id,string"` tag for JSON string marshaling
- Nullable fields use pointers: `*string`, `*time.Time`, `*int64`
- `PasswordHash` field on User has `json:"-"` to never serialize
- `StorageKey` on Attachment has `json:"-"` (internal only); `URL` is computed for client responses
- `MessageWithAuthor` embeds `Message` and adds denormalized author info + attachments
