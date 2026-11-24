package service

import (
	"context"
	"github.com/victorivanov/retrocast/internal/permissions"
	"strconv"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// RoleService handles role and channel override business logic.
type RoleService struct {
	guilds    database.GuildRepository
	roles     database.RoleRepository
	members   database.MemberRepository
	channels  database.ChannelRepository
	overrides database.ChannelOverrideRepository
	snowflake *snowflake.Generator
	gateway   gateway.Dispatcher
	perms     *PermissionChecker
}

// NewRoleService creates a RoleService.
func NewRoleService(
	guilds database.GuildRepository,
	roles database.RoleRepository,
	members database.MemberRepository,
	channels database.ChannelRepository,
	overrides database.ChannelOverrideRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *RoleService {
	return &RoleService{
		guilds:    guilds,
		roles:     roles,
		members:   members,
		channels:  channels,
		overrides: overrides,
		snowflake: sf,
		gateway:   gw,
		perms:     perms,
	}
}

// CreateRole creates a new role in a guild with role hierarchy enforcement.
func (s *RoleService) CreateRole(ctx context.Context, guildID, actorID int64, name string, color int, permBits int64, position int) (*models.Role, error) {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, actorID, permissions.PermManageRoles); err != nil {
		return nil, err
	}
	if err := s.perms.RequireGrantPermissions(ctx, guildID, actorID, permBits); err != nil {
		return nil, err
	}

	if position < 0 {
		return nil, BadRequest("INVALID_POSITION", "position must be non-negative")
	}
	if name == "" || len(name) > 100 {
		return nil, BadRequest("INVALID_NAME", "name must be 1-100 characters")
	}

	isOwner, err := s.perms.IsGuildOwner(ctx, guildID, actorID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		highest, err := s.perms.HighestRolePosition(ctx, guildID, actorID)
		if err != nil {
			return nil, err
		}
		if position >= highest {
			return nil, RoleHierarchyError("cannot create a role at or above your highest role position")
		}
	}

	role := &models.Role{
		ID:          s.snowflake.Generate().Int64(),
		GuildID:     guildID,
		Name:        name,
		Color:       color,
		Permissions: permBits,
		Position:    position,
	}

	if err := s.roles.Create(ctx, role); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleCreate, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role": role})
	return role, nil
}

// ListRoles returns all roles for a guild.
func (s *RoleService) ListRoles(ctx context.Context, guildID, actorID int64) ([]models.Role, error) {
	member, err := s.members.GetByGuildAndUser(ctx, guildID, actorID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	roles, err := s.roles.GetByGuildID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if roles == nil {
		roles = []models.Role{}
	}
	return roles, nil
}

// UpdateRole updates a role with hierarchy enforcement.
func (s *RoleService) UpdateRole(ctx context.Context, guildID, actorID, roleID int64, name *string, color *int, permBits *int64, position *int) (*models.Role, error) {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, actorID, permissions.PermManageRoles); err != nil {
		return nil, err
	}
	if position != nil && *position < 0 {
		return nil, BadRequest("INVALID_POSITION", "position must be non-negative")
	}
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if role == nil || role.GuildID != guildID {
		return nil, NotFound("NOT_FOUND", "role not found")
	}

	if role.IsDefault && position != nil && *position != 0 {
		return nil, BadRequest("INVALID_POSITION", "the @everyone role must stay at position 0")
	}
	effectiveBits := role.Permissions
	if permBits != nil {
		effectiveBits |= *permBits
	}
	if err := s.perms.RequireGrantPermissions(ctx, guildID, actorID, effectiveBits); err != nil {
		return nil, err
	}

	isOwner, err := s.perms.IsGuildOwner(ctx, guildID, actorID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		highest, err := s.perms.HighestRolePosition(ctx, guildID, actorID)
		if err != nil {
			return nil, err
		}
		if role.Position >= highest || (position != nil && *position >= highest) {
			return nil, RoleHierarchyError("cannot modify a role at or above your highest role position")
		}
	}

	if name != nil {
		if *name == "" || len(*name) > 100 {
			return nil, BadRequest("INVALID_NAME", "name must be 1-100 characters")
		}
		role.Name = *name
	}
	if color != nil {
		role.Color = *color
	}
	if permBits != nil {
		role.Permissions = *permBits
	}
	if position != nil {
		role.Position = *position
	}

	if err := s.roles.Update(ctx, role); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleUpdate, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role": role})
	return role, nil
}

// DeleteRole deletes a role with hierarchy enforcement.
func (s *RoleService) DeleteRole(ctx context.Context, guildID, actorID, roleID int64) error {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, actorID, permissions.PermManageRoles); err != nil {
		return err
	}
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if role == nil || role.GuildID != guildID {
		return NotFound("NOT_FOUND", "role not found")
	}
	if role.IsDefault {
		return Forbidden("CANNOT_DELETE", "cannot delete the @everyone role")
	}

	if err := s.perms.RequireGrantPermissions(ctx, guildID, actorID, role.Permissions); err != nil {
		return err
	}

	isOwner, err := s.perms.IsGuildOwner(ctx, guildID, actorID)
	if err != nil {
		return err
	}
	if !isOwner {
		highest, err := s.perms.HighestRolePosition(ctx, guildID, actorID)
		if err != nil {
			return err
		}
		if role.Position >= highest {
			return RoleHierarchyError("cannot delete a role at or above your highest role position")
		}
	}

	if err := s.roles.Delete(ctx, roleID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleDelete, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role_id": strconv.FormatInt(roleID, 10)})
	return nil
}

// AssignRole assigns a role to a member with hierarchy enforcement.
func (s *RoleService) AssignRole(ctx context.Context, guildID, actorID, userID, roleID int64) error {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, actorID, permissions.PermManageRoles); err != nil {
		return err
	}
	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return NotFound("NOT_FOUND", "member not found")
	}

	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if role == nil || role.GuildID != guildID {
		return NotFound("NOT_FOUND", "role not found")
	}

	if err := s.perms.RequireGrantPermissions(ctx, guildID, actorID, role.Permissions); err != nil {
		return err
	}

	isOwner, err := s.perms.IsGuildOwner(ctx, guildID, actorID)
	if err != nil {
		return err
	}
	if !isOwner {
		highest, err := s.perms.HighestRolePosition(ctx, guildID, actorID)
		if err != nil {
			return err
		}
		if role.Position >= highest {
			return RoleHierarchyError("cannot assign a role at or above your highest role position")
		}
	}

	if err := s.members.AddRole(ctx, guildID, userID, roleID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	return nil
}

// RemoveRole removes a role from a member with hierarchy enforcement.
func (s *RoleService) RemoveRole(ctx context.Context, guildID, actorID, userID, roleID int64) error {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, actorID, permissions.PermManageRoles); err != nil {
		return err
	}
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if role == nil || role.GuildID != guildID {
		return NotFound("NOT_FOUND", "role not found")
	}

	if err := s.perms.RequireGrantPermissions(ctx, guildID, actorID, role.Permissions); err != nil {
		return err
	}

	isOwner, err := s.perms.IsGuildOwner(ctx, guildID, actorID)
	if err != nil {
		return err
	}
	if !isOwner {
		role, err := s.roles.GetByID(ctx, roleID)
		if err != nil {
			return Internal("INTERNAL", "internal server error")
		}
		if role != nil {
			highest, err := s.perms.HighestRolePosition(ctx, guildID, actorID)
			if err != nil {
				return err
			}
			if role.Position >= highest {
				return RoleHierarchyError("cannot remove a role at or above your highest role position")
			}
		}
	}

	if err := s.members.RemoveRole(ctx, guildID, userID, roleID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	return nil
}

// SetChannelOverride creates or updates a channel permission override.
func (s *RoleService) SetChannelOverride(ctx context.Context, channelID, actorID, roleID, allow, deny int64) (*models.ChannelOverride, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if ch == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	if err := s.checkOverride(ctx, ch, actorID, roleID); err != nil {
		return nil, err
	}

	if err := s.perms.RequireGrantPermissions(ctx, ch.GuildID, actorID, allow); err != nil {
		return nil, err
	}

	override := &models.ChannelOverride{
		ChannelID: channelID,
		RoleID:    roleID,
		Allow:     allow,
		Deny:      deny,
	}

	if err := s.overrides.Set(ctx, override); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return override, nil
}

// DeleteChannelOverride removes a channel permission override.
func (s *RoleService) DeleteChannelOverride(ctx context.Context, channelID, actorID, roleID int64) error {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if ch == nil {
		return NotFound("NOT_FOUND", "channel not found")
	}
	if err := s.checkOverride(ctx, ch, actorID, roleID); err != nil {
		return err
	}

	if err := s.overrides.Delete(ctx, channelID, roleID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	return nil
}

func (s *RoleService) checkOverride(ctx context.Context, ch *models.Channel, actorID, roleID int64) error {
	if err := s.perms.RequireChannelPermission(ctx, ch.GuildID, ch.ID, actorID, permissions.PermManageRoles); err != nil {
		return err
	}
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if role == nil || role.GuildID != ch.GuildID {
		return NotFound("NOT_FOUND", "role not found")
	}
	return nil
}
