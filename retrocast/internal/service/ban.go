package service

import (
	"context"
	"time"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// BanService handles ban/unban business logic.
type BanService struct {
	guilds  database.GuildRepository
	members database.MemberRepository
	roles   database.RoleRepository
	bans    database.BanRepository
	gateway gateway.Dispatcher
	perms   *PermissionChecker
}

// NewBanService creates a BanService.
func NewBanService(
	guilds database.GuildRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	bans database.BanRepository,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *BanService {
	return &BanService{
		guilds:  guilds,
		members: members,
		roles:   roles,
		bans:    bans,
		gateway: gw,
		perms:   perms,
	}
}

// BanMember bans a user from a guild with role hierarchy enforcement.
func (s *BanService) BanMember(ctx context.Context, guildID, callerID, targetUserID int64, reason *string) error {
	if callerID == targetUserID {
		return BadRequest("CANNOT_BAN_SELF", "you cannot ban yourself")
	}

	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, callerID, permissions.PermBanMembers); err != nil {
		return err
	}

	guild, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return NotFound("NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == targetUserID {
		return Forbidden("FORBIDDEN", "cannot ban the guild owner")
	}

	callerRoles, err := s.roles.GetByMember(ctx, guildID, callerID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	targetRoles, err := s.roles.GetByMember(ctx, guildID, targetUserID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	if guild.OwnerID != callerID && highestPosition(targetRoles) >= highestPosition(callerRoles) {
		return RoleHierarchyError("your highest role must be above the target's highest role")
	}

	ban := &models.Ban{
		GuildID:   guildID,
		UserID:    targetUserID,
		Reason:    reason,
		CreatedBy: callerID,
		CreatedAt: time.Now(),
	}

	if err := s.bans.Create(ctx, ban); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	_ = s.members.Delete(ctx, guildID, targetUserID)

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildBanAdd, ban)
	s.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberRemove, map[string]any{"guild_id": guildID, "user_id": targetUserID})

	return nil
}

// UnbanMember removes a ban from a guild.
func (s *BanService) UnbanMember(ctx context.Context, guildID, callerID, targetUserID int64) error {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, callerID, permissions.PermBanMembers); err != nil {
		return err
	}

	if err := s.bans.Delete(ctx, guildID, targetUserID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildBanRemove, map[string]any{"guild_id": guildID, "user_id": targetUserID})

	return nil
}

// ListBans returns all bans for a guild.
func (s *BanService) ListBans(ctx context.Context, guildID, callerID int64) ([]models.Ban, error) {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, callerID, permissions.PermBanMembers); err != nil {
		return nil, err
	}

	bans, err := s.bans.GetByGuildID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if bans == nil {
		bans = []models.Ban{}
	}
	return bans, nil
}

// highestPosition returns the highest role position from a list of roles.
func highestPosition(roles []models.Role) int {
	max := -1
	for _, r := range roles {
		if r.Position > max {
			max = r.Position
		}
	}
	return max
}
