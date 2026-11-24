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
func (s *RoleService) createRole(ctx context.Context, guildID, actorID int64, name string, color int, permBits int64, position int) (*models.Role, error) {
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
func (s *RoleService) updateRole(ctx context.Context, guildID, actorID, roleID int64, name *string, color *int, permBits *int64, position *int) (*models.Role, error) {
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

	return role, nil
}

// DeleteRole deletes a role with hierarchy enforcement.
func (s *RoleService) deleteRole(ctx context.Context, guildID, actorID, roleID int64) error {
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

	s.invalidateGateway(guildID)
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

	s.invalidateGateway(guildID)
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

	s.invalidateGateway(ch.GuildID)
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
	s.invalidateGateway(ch.GuildID)
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

func (s *RoleService) invalidateGateway(guildID int64) {
	if g, ok := s.gateway.(interface{ InvalidateGuildPermissions(int64) }); ok {
		g.InvalidateGuildPermissions(guildID)
	}
}

// RolePosition is the complete optimistic snapshot sent by a reorder client.
type RolePosition struct {
	ID       int64 `json:"id,string"`
	Position int   `json:"position"`
}

// ReorderRoles serializes role writes and checks the snapshot before any update.
func (s *RoleService) ReorderRoles(ctx context.Context, guildID, actorID int64, expected []RolePosition, order []string) ([]models.Role, error) {
	var result []models.Role
	var changed []models.Role
	err := database.InTransaction(ctx, s.roles, func(ctx context.Context) error {
		if err := database.LockGuildForModeration(ctx, s.roles, guildID); err != nil {
			return err
		}
		if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, actorID, permissions.PermManageRoles); err != nil {
			return err
		}
		roles, err := s.roles.GetByGuildID(ctx, guildID)
		if err != nil {
			return err
		}
		snapshot := make(map[int64]int, len(expected))
		for _, r := range expected {
			if _, ok := snapshot[r.ID]; ok {
				return BadRequest("INVALID_ORDER", "duplicate snapshot role")
			}
			snapshot[r.ID] = r.Position
		}
		if len(snapshot) != len(roles) {
			return Conflict("STALE_ROLES", "Roles changed; reload and try again")
		}
		byID := make(map[int64]models.Role, len(roles))
		movable := 0
		for _, r := range roles {
			if pos, ok := snapshot[r.ID]; !ok || pos != r.Position {
				return Conflict("STALE_ROLES", "Roles changed; reload and try again")
			}
			byID[r.ID] = r
			if !r.IsDefault {
				movable++
			}
		}
		if len(order) != movable {
			return BadRequest("INVALID_ORDER", "include every non-default role exactly once")
		}
		owner, err := s.perms.IsGuildOwner(ctx, guildID, actorID)
		if err != nil {
			return err
		}
		highest, err := s.perms.HighestRolePosition(ctx, guildID, actorID)
		if err != nil {
			return err
		}
		seen := make(map[int64]bool)
		next := 1
		last := 0
		for _, raw := range order {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return BadRequest("INVALID_ORDER", "invalid role id")
			}
			r, ok := byID[id]
			if !ok || seen[id] || r.IsDefault {
				return BadRequest("INVALID_ORDER", "unknown, duplicate or default role")
			}
			seen[id] = true
			position := next
			if !owner && r.Position >= highest {
				position = r.Position
			}
			if !owner && r.Position < highest && position >= highest {
				return RoleHierarchyError("cannot move a role at or above your highest role position")
			}
			if position < last {
				return RoleHierarchyError("cannot reorder roles at or above your highest role position")
			}
			if position != r.Position {
				if !owner && r.Position >= highest {
					return RoleHierarchyError("cannot modify a role at or above your highest role position")
				}
				if err := s.perms.RequireGrantPermissions(ctx, guildID, actorID, r.Permissions); err != nil {
					return err
				}
				r.Position = position
				changed = append(changed, r)
			}
			last = position
			next = position + 1
		}
		// Validate the entire operation before writing; the transaction rolls back failures.
		for i := range changed {
			if err := s.roles.Update(ctx, &changed[i]); err != nil {
				return err
			}
			byID[changed[i].ID] = changed[i]
		}
		for _, r := range roles {
			result = append(result, byID[r.ID])
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, r := range changed {
		s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleUpdate, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role": r})
	}
	return result, nil
}

func (s *RoleService) CreateRole(ctx context.Context, guildID, actorID int64, name string, color int, permBits int64, position int) (*models.Role, error) {
	var result *models.Role
	err := database.InTransaction(ctx, s.roles, func(ctx context.Context) error {
		if err := database.LockGuildForModeration(ctx, s.roles, guildID); err != nil {
			return err
		}
		var err error
		result, err = s.createRole(ctx, guildID, actorID, name, color, permBits, position)
		return err
	})
	if err == nil {
		s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleCreate, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role": result})
	}
	return result, err
}

func (s *RoleService) UpdateRole(ctx context.Context, guildID, actorID, roleID int64, name *string, color *int, permBits *int64, position *int) (*models.Role, error) {
	var result *models.Role
	err := database.InTransaction(ctx, s.roles, func(ctx context.Context) error {
		if err := database.LockGuildForModeration(ctx, s.roles, guildID); err != nil {
			return err
		}
		var err error
		result, err = s.updateRole(ctx, guildID, actorID, roleID, name, color, permBits, position)
		return err
	})
	if err == nil {
		s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleUpdate, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role": result})
	}
	return result, err
}

func (s *RoleService) DeleteRole(ctx context.Context, guildID, actorID, roleID int64) error {
	err := database.InTransaction(ctx, s.roles, func(ctx context.Context) error {
		if err := database.LockGuildForModeration(ctx, s.roles, guildID); err != nil {
			return err
		}
		return s.deleteRole(ctx, guildID, actorID, roleID)
	})
	if err == nil {
		s.gateway.DispatchToGuild(guildID, gateway.EventGuildRoleDelete, map[string]any{"guild_id": strconv.FormatInt(guildID, 10), "role_id": strconv.FormatInt(roleID, 10)})
	}
	return err
}
