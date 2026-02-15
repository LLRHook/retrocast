package permissions

import "strings"

// Permission is a bitfield representing a set of permissions.
type Permission int64

const (
	PermViewChannel        Permission = 1 << 0
	PermSendMessages       Permission = 1 << 1
	PermManageMessages     Permission = 1 << 2
	PermManageChannels     Permission = 1 << 3
	PermManageRoles        Permission = 1 << 4
	PermKickMembers        Permission = 1 << 5
	PermBanMembers         Permission = 1 << 6
	PermManageGuild        Permission = 1 << 7
	PermConnect            Permission = 1 << 8  // voice
	PermSpeak              Permission = 1 << 9  // voice
	PermMuteMembers        Permission = 1 << 10 // voice
	PermDeafenMembers      Permission = 1 << 11 // voice
	PermMoveMembers        Permission = 1 << 12 // voice
	PermMentionEveryone    Permission = 1 << 13
	PermAttachFiles        Permission = 1 << 14
	PermReadMessageHistory Permission = 1 << 15
	PermCreateInvite       Permission = 1 << 16
	PermChangeNickname     Permission = 1 << 17
	PermManageNicknames    Permission = 1 << 18
	PermAdministrator      Permission = 1 << 31 // bypasses all checks

	// Convenience sets
	PermAllText  = PermViewChannel | PermSendMessages | PermManageMessages | PermReadMessageHistory | PermMentionEveryone | PermAttachFiles
	PermAllVoice = PermConnect | PermSpeak | PermMuteMembers | PermDeafenMembers | PermMoveMembers
	PermAll      = Permission(0x7FFFFFFFFFFFFFFF)
)

// Has returns true if p contains all bits in perm.
func (p Permission) Has(perm Permission) bool { return p&perm == perm }

// Add returns p with the bits from perm set.
func (p Permission) Add(perm Permission) Permission { return p | perm }

// Remove returns p with the bits from perm cleared.
func (p Permission) Remove(perm Permission) Permission { return p &^ perm }

// DefaultEveryonePerms is the default permission set for the @everyone role.
var DefaultEveryonePerms = PermViewChannel | PermSendMessages | PermReadMessageHistory | PermConnect | PermSpeak | PermCreateInvite | PermChangeNickname

// permNames maps individual permission bits to their string names.
var permNames = map[Permission]string{
	PermViewChannel:        "VIEW_CHANNEL",
	PermSendMessages:       "SEND_MESSAGES",
	PermManageMessages:     "MANAGE_MESSAGES",
	PermManageChannels:     "MANAGE_CHANNELS",
	PermManageRoles:        "MANAGE_ROLES",
	PermKickMembers:        "KICK_MEMBERS",
	PermBanMembers:         "BAN_MEMBERS",
	PermManageGuild:        "MANAGE_GUILD",
	PermConnect:            "CONNECT",
	PermSpeak:              "SPEAK",
	PermMuteMembers:        "MUTE_MEMBERS",
	PermDeafenMembers:      "DEAFEN_MEMBERS",
	PermMoveMembers:        "MOVE_MEMBERS",
	PermMentionEveryone:    "MENTION_EVERYONE",
	PermAttachFiles:        "ATTACH_FILES",
	PermReadMessageHistory: "READ_MESSAGE_HISTORY",
	PermCreateInvite:       "CREATE_INVITE",
	PermChangeNickname:     "CHANGE_NICKNAME",
	PermManageNicknames:    "MANAGE_NICKNAMES",
	PermAdministrator:      "ADMINISTRATOR",
}

// String returns a human-readable representation of the permission set,
// listing all set permission names separated by " | ".
func (p Permission) String() string {
	if p == 0 {
		return "NONE"
	}

	var names []string
	for bit, name := range permNames {
		if p.Has(bit) {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return "UNKNOWN"
	}
	return strings.Join(names, " | ")
}
