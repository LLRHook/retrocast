package service

import (
	"context"

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

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleCreate, map[string]any{"guild_id": guildID, "role": role})
	return role, nil
}

// ListRoles returns all roles for a guild.
func (s *RoleService) ListRoles(ctx context.Context, guildID int64) ([]models.Role, error) {
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
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if role == nil {
		return nil, NotFound("NOT_FOUND", "role not found")
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
		if role.Position >= highest {
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

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleUpdate, map[string]any{"guild_id": guildID, "role": role})
	return role, nil
}

// DeleteRole deletes a role with hierarchy enforcement.
func (s *RoleService) DeleteRole(ctx context.Context, guildID, actorID, roleID int64) error {
	role, err := s.roles.GetByID(ctx, roleID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if role == nil {
		return NotFound("NOT_FOUND", "role not found")
	}
	if role.IsDefault {
		return Forbidden("CANNOT_DELETE", "cannot delete the @everyone role")
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

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleDelete, map[string]any{"guild_id": guildID, "role_id": roleID})
	return nil
}

// AssignRole assigns a role to a member with hierarchy enforcement.
func (s *RoleService) AssignRole(ctx context.Context, guildID, actorID, userID, roleID int64) error {
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
func (s *RoleService) SetChannelOverride(ctx context.Context, channelID, roleID, allow, deny int64) (*models.ChannelOverride, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if ch == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
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
func (s *RoleService) DeleteChannelOverride(ctx context.Context, channelID, roleID int64) error {
	if err := s.overrides.Delete(ctx, channelID, roleID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	return nil
}
