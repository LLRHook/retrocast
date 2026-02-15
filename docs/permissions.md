# Permissions

Retrocast uses a two-level bitfield RBAC system modeled after Discord's permission system. Permissions are stored as `int64` bitmasks on roles, with per-channel overrides for fine-grained control.

Source: `retrocast/internal/permissions/`

## Permission Bits

19 individual permission bits defined in `bitfield.go`:

| Bit | Value | Constant | Category | Description |
|-----|-------|----------|----------|-------------|
| 0 | `1 << 0` | `PermViewChannel` | Text | View channel and read messages |
| 1 | `1 << 1` | `PermSendMessages` | Text | Send messages in text channels |
| 2 | `1 << 2` | `PermManageMessages` | Text | Delete others' messages |
| 3 | `1 << 3` | `PermManageChannels` | Server | Create, edit, delete channels |
| 4 | `1 << 4` | `PermManageRoles` | Server | Create, edit, delete roles |
| 5 | `1 << 5` | `PermKickMembers` | Mod | Kick members from guild |
| 6 | `1 << 6` | `PermBanMembers` | Mod | Ban members from guild |
| 7 | `1 << 7` | `PermManageGuild` | Server | Edit guild name, icon, settings |
| 8 | `1 << 8` | `PermConnect` | Voice | Connect to voice channels |
| 9 | `1 << 9` | `PermSpeak` | Voice | Speak in voice channels |
| 10 | `1 << 10` | `PermMuteMembers` | Voice | Server-mute other members |
| 11 | `1 << 11` | `PermDeafenMembers` | Voice | Server-deafen other members |
| 12 | `1 << 12` | `PermMoveMembers` | Voice | Move members between voice channels |
| 13 | `1 << 13` | `PermMentionEveryone` | Text | Use @everyone and @here |
| 14 | `1 << 14` | `PermAttachFiles` | Text | Upload files/images |
| 15 | `1 << 15` | `PermReadMessageHistory` | Text | View message history |
| 16 | `1 << 16` | `PermCreateInvite` | Server | Create invite links |
| 17 | `1 << 17` | `PermChangeNickname` | Server | Change own nickname |
| 18 | `1 << 18` | `PermManageNicknames` | Server | Change others' nicknames |
| 31 | `1 << 31` | `PermAdministrator` | Special | Bypasses ALL permission checks |

## Convenience Sets

```go
PermAllText  = PermViewChannel | PermSendMessages | PermManageMessages |
               PermReadMessageHistory | PermMentionEveryone | PermAttachFiles

PermAllVoice = PermConnect | PermSpeak | PermMuteMembers |
               PermDeafenMembers | PermMoveMembers

PermAll      = 0x7FFFFFFFFFFFFFFF  // every bit set
```

## Default @everyone Permissions

When a guild is created, the `@everyone` role (with `is_default = true`) gets:

```go
DefaultEveryonePerms = PermViewChannel | PermSendMessages | PermReadMessageHistory |
                       PermConnect | PermSpeak | PermCreateInvite | PermChangeNickname
```

Value: `0x000000000001_0307` (bits 0, 1, 8, 9, 15, 16, 17)

## Bitfield Operations

The `Permission` type provides three methods:

```go
func (p Permission) Has(perm Permission) bool     // p & perm == perm
func (p Permission) Add(perm Permission) Permission    // p | perm
func (p Permission) Remove(perm Permission) Permission // p &^ perm
```

To check if a user can send messages: `userPerms.Has(PermSendMessages)`

## Permission Resolution Algorithm

Two-phase computation defined in `resolver.go`:

### Phase 1: Base Permissions (`ComputeBasePermissions`)

Computes guild-level permissions for a member:

1. Start with the `@everyone` role's `permissions` bitfield
2. OR all the member's explicitly assigned role `permissions` bitfields
3. If the result includes `PermAdministrator`, return `PermAll` (bypass everything)

```go
func ComputeBasePermissions(everyoneRole models.Role, memberRoles []models.Role) Permission {
    perms := Permission(everyoneRole.Permissions)
    for _, role := range memberRoles {
        perms = perms.Add(Permission(role.Permissions))
    }
    if perms.Has(PermAdministrator) {
        return PermAll
    }
    return perms
}
```

### Phase 2: Channel Permissions (`ComputeChannelPermissions`)

Applies channel-specific overrides on top of base permissions:

1. If base permissions include `PermAdministrator`, return `PermAll` (skip overrides)
2. Apply `@everyone` channel override: **deny** first, then **allow**
3. Aggregate all role overrides: OR all `allow` fields together, OR all `deny` fields together
4. Apply aggregated role overrides: **deny** first, then **allow**
5. Return final permissions

```go
func ComputeChannelPermissions(basePerms Permission,
    everyoneOverride *models.ChannelOverride,
    roleOverrides []models.ChannelOverride) Permission {

    if basePerms.Has(PermAdministrator) {
        return PermAll
    }
    perms := basePerms

    // @everyone override
    if everyoneOverride != nil {
        perms = perms.Remove(Permission(everyoneOverride.Deny))
        perms = perms.Add(Permission(everyoneOverride.Allow))
    }

    // Role overrides (aggregated)
    var roleAllow, roleDeny Permission
    for _, o := range roleOverrides {
        roleAllow = roleAllow.Add(Permission(o.Allow))
        roleDeny = roleDeny.Add(Permission(o.Deny))
    }
    perms = perms.Remove(roleDeny)
    perms = perms.Add(roleAllow)

    return perms
}
```

## Override Precedence

The override order matters:

1. `@everyone` role override is applied first (affects the base for everyone)
2. All other role overrides are aggregated and applied second (role-specific adjustments)
3. Within each step: deny is applied before allow (allow can re-grant denied permissions)

This means: if a user's "Moderator" role allows `MANAGE_MESSAGES` in a channel, it overrides a deny from the `@everyone` override for that channel.

## Database Storage

Channel overrides are stored in the `channel_overrides` table:

```sql
CREATE TABLE channel_overrides (
    channel_id  BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    role_id     BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    allow_perms BIGINT NOT NULL DEFAULT 0,
    deny_perms  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (channel_id, role_id)
);
```

Repository: `ChannelOverrideRepository` with `Set`, `GetByChannel`, and `Delete` methods.

## Permission Checks in Handlers

### Guild-Level

The `GuildHandler` exposes a `RequirePermission()` middleware factory. Other handlers (channel, member) receive this middleware and use it to guard routes.

### Channel-Level

The `MessageHandler` and related handlers use the `permissions.Resolver` to compute channel-level permissions by:

1. Fetching the member's roles from the database
2. Fetching the `@everyone` role for the guild
3. Computing base permissions
4. Fetching channel overrides
5. Computing final channel permissions
6. Checking the required bit

### Guild Owner Bypass

The guild owner always has full permissions (equivalent to `PermAdministrator`), regardless of their actual role assignments. This is checked in handler logic before the bitfield computation.

## String Representation

`Permission.String()` returns a human-readable format:

```
PermSendMessages | PermViewChannel    -> "SEND_MESSAGES | VIEW_CHANNEL"
Permission(0)                         -> "NONE"
```

All 19 permission names are mapped in the `permNames` variable in `bitfield.go`.
