package service

import (
	"context"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// PermissionChecker provides guild-level and channel-level permission checks.
type PermissionChecker struct {
	guilds    database.GuildRepository
	members   database.MemberRepository
	roles     database.RoleRepository
	overrides database.ChannelOverrideRepository
}

// NewPermissionChecker creates a PermissionChecker.
func NewPermissionChecker(
	guilds database.GuildRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	overrides database.ChannelOverrideRepository,
) *PermissionChecker {
	return &PermissionChecker{
		guilds:    guilds,
		members:   members,
		roles:     roles,
		overrides: overrides,
	}
}

// RequireGuildPermission checks that the user has the given permission in a guild.
// Guild owners and administrators bypass all checks.
func (p *PermissionChecker) RequireGuildPermission(ctx context.Context, guildID, userID, perm int64) error {
	guild, err := p.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return NotFound("NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == userID {
		return nil
	}

	member, err := p.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return NotFound("NOT_FOUND", "guild not found")
	}

	roles, err := p.roles.GetByMember(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	var perms int64
	for _, r := range roles {
		perms |= r.Permissions
	}

	if perms&int64(permissions.PermAdministrator) != 0 {
		return nil
	}

	if perms&perm == 0 {
		return Forbidden("MISSING_PERMISSIONS", "you do not have permission to perform this action")
	}
	return nil
}

// RequireChannelPermission checks that the user has the given permission in a channel,
// applying channel-level overrides on top of guild-level base permissions.
func (p *PermissionChecker) RequireChannelPermission(ctx context.Context, guildID, channelID, userID int64, perm permissions.Permission) error {
	guild, err := p.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return NotFound("NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == userID {
		return nil
	}

	member, err := p.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return Forbidden("FORBIDDEN", "you are not a member of this guild")
	}

	memberRoles, err := p.roles.GetByMember(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	allRoles, err := p.roles.GetByGuildID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	var everyoneRole models.Role
	for _, r := range allRoles {
		if r.IsDefault {
			everyoneRole = r
			break
		}
	}

	basePerms := permissions.ComputeBasePermissions(everyoneRole, memberRoles)

	channelOverrides, err := p.overrides.GetByChannel(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	var everyoneOverride *models.ChannelOverride
	var roleOverrides []models.ChannelOverride

	memberRoleIDs := make(map[int64]bool, len(memberRoles))
	for _, r := range memberRoles {
		memberRoleIDs[r.ID] = true
	}

	for i := range channelOverrides {
		if channelOverrides[i].RoleID == everyoneRole.ID {
			everyoneOverride = &channelOverrides[i]
		} else if memberRoleIDs[channelOverrides[i].RoleID] {
			roleOverrides = append(roleOverrides, channelOverrides[i])
		}
	}

	computed := permissions.ComputeChannelPermissions(basePerms, everyoneOverride, roleOverrides)
	if !computed.Has(perm) {
		return Forbidden("MISSING_PERMISSIONS", "you do not have the required permissions")
	}

	return nil
}

// RequireGuildPermissionByPerm is like RequireGuildPermission but uses permissions.Permission type.
// Used by invite/ban handlers that pass permissions.Permission instead of raw int64.
func (p *PermissionChecker) RequireGuildPermissionByPerm(ctx context.Context, guildID, userID int64, perm permissions.Permission) error {
	guild, err := p.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return NotFound("NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == userID {
		return nil
	}

	member, err := p.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return Forbidden("FORBIDDEN", "you are not a member of this guild")
	}

	memberRoles, err := p.roles.GetByMember(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	allRoles, err := p.roles.GetByGuildID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	var everyoneRole models.Role
	for _, r := range allRoles {
		if r.IsDefault {
			everyoneRole = r
			break
		}
	}

	computed := permissions.ComputeBasePermissions(everyoneRole, memberRoles)
	if !computed.Has(perm) {
		return Forbidden("MISSING_PERMISSIONS", "you do not have the required permissions")
	}

	return nil
}

// IsGuildOwner returns true if the user is the owner of the guild.
func (p *PermissionChecker) IsGuildOwner(ctx context.Context, guildID, userID int64) (bool, error) {
	guild, err := p.guilds.GetByID(ctx, guildID)
	if err != nil {
		return false, Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return false, nil
	}
	return guild.OwnerID == userID, nil
}

// HighestRolePosition returns the highest position among the user's roles.
func (p *PermissionChecker) HighestRolePosition(ctx context.Context, guildID, userID int64) (int, error) {
	memberRoles, err := p.roles.GetByMember(ctx, guildID, userID)
	if err != nil {
		return 0, Internal("INTERNAL", "internal server error")
	}
	highest := 0
	for _, r := range memberRoles {
		if r.Position > highest {
			highest = r.Position
		}
	}
	return highest, nil
}
