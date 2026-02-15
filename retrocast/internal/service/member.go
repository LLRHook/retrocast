package service

import (
	"context"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// MemberService handles member management business logic.
type MemberService struct {
	members database.MemberRepository
	guilds  database.GuildRepository
	roles   database.RoleRepository
	gateway gateway.Dispatcher
	perms   *PermissionChecker
}

// NewMemberService creates a MemberService.
func NewMemberService(
	members database.MemberRepository,
	guilds database.GuildRepository,
	roles database.RoleRepository,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *MemberService {
	return &MemberService{
		members: members,
		guilds:  guilds,
		roles:   roles,
		gateway: gw,
		perms:   perms,
	}
}

// ListMembers returns members of a guild. Caller must be a member.
func (s *MemberService) ListMembers(ctx context.Context, guildID, userID int64, limit, offset int) ([]models.Member, error) {
	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	members, err := s.members.GetByGuildID(ctx, guildID, limit, offset)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if members == nil {
		members = []models.Member{}
	}
	return members, nil
}

// GetMember returns a specific member. Caller must be a member.
func (s *MemberService) GetMember(ctx context.Context, guildID, callerID, targetUserID int64) (*models.Member, error) {
	callerMember, err := s.members.GetByGuildAndUser(ctx, guildID, callerID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if callerMember == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	member, err := s.members.GetByGuildAndUser(ctx, guildID, targetUserID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "member not found")
	}

	return member, nil
}

// UpdateSelf updates the caller's own member profile (nickname).
func (s *MemberService) UpdateSelf(ctx context.Context, guildID, userID int64, nickname *string) (*models.Member, error) {
	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	if nickname != nil {
		if len(*nickname) > 32 {
			return nil, BadRequest("INVALID_NICKNAME", "nickname must be 32 characters or fewer")
		}
		if *nickname == "" {
			member.Nickname = nil
		} else {
			member.Nickname = nickname
		}
	}

	if err := s.members.Update(ctx, member); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberUpdate, member)
	return member, nil
}

// UpdateMember updates another member's nickname and/or roles.
func (s *MemberService) UpdateMember(ctx context.Context, guildID, callerID, targetUserID int64, nickname *string, roleIDs *[]int64) (*models.Member, error) {
	if nickname != nil {
		if err := s.perms.RequireGuildPermission(ctx, guildID, callerID, int64(permissions.PermManageNicknames)); err != nil {
			return nil, err
		}
	}
	if roleIDs != nil {
		if err := s.perms.RequireGuildPermission(ctx, guildID, callerID, int64(permissions.PermManageRoles)); err != nil {
			return nil, err
		}
	}

	member, err := s.members.GetByGuildAndUser(ctx, guildID, targetUserID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "member not found")
	}

	if nickname != nil {
		if len(*nickname) > 32 {
			return nil, BadRequest("INVALID_NICKNAME", "nickname must be 32 characters or fewer")
		}
		if *nickname == "" {
			member.Nickname = nil
		} else {
			member.Nickname = nickname
		}
		if err := s.members.Update(ctx, member); err != nil {
			return nil, Internal("INTERNAL", "internal server error")
		}
	}

	if roleIDs != nil {
		guildRoles, err := s.roles.GetByGuildID(ctx, guildID)
		if err != nil {
			return nil, Internal("INTERNAL", "internal server error")
		}

		validRoles := make(map[int64]bool, len(guildRoles))
		for _, r := range guildRoles {
			if !r.IsDefault {
				validRoles[r.ID] = true
			}
		}

		for _, roleID := range member.Roles {
			if validRoles[roleID] {
				if err := s.members.RemoveRole(ctx, guildID, targetUserID, roleID); err != nil {
					return nil, Internal("INTERNAL", "internal server error")
				}
			}
		}

		for _, roleID := range *roleIDs {
			if !validRoles[roleID] {
				return nil, BadRequest("INVALID_ROLE", "invalid role ID")
			}
			if err := s.members.AddRole(ctx, guildID, targetUserID, roleID); err != nil {
				return nil, Internal("INTERNAL", "internal server error")
			}
		}

		member, err = s.members.GetByGuildAndUser(ctx, guildID, targetUserID)
		if err != nil {
			return nil, Internal("INTERNAL", "internal server error")
		}
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberUpdate, member)
	return member, nil
}

// KickMember removes a member from the guild.
func (s *MemberService) KickMember(ctx context.Context, guildID, callerID, targetUserID int64) error {
	if err := s.perms.RequireGuildPermission(ctx, guildID, callerID, int64(permissions.PermKickMembers)); err != nil {
		return err
	}

	guild, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild != nil && guild.OwnerID == targetUserID {
		return Forbidden("FORBIDDEN", "cannot kick the guild owner")
	}

	member, err := s.members.GetByGuildAndUser(ctx, guildID, targetUserID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return NotFound("NOT_FOUND", "member not found")
	}

	if err := s.members.Delete(ctx, guildID, targetUserID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberRemove, map[string]any{"guild_id": guildID, "user_id": targetUserID})
	return nil
}

// LeaveGuild allows a member to leave a guild. The owner cannot leave.
func (s *MemberService) LeaveGuild(ctx context.Context, guildID, userID int64) error {
	guild, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild != nil && guild.OwnerID == userID {
		return Forbidden("FORBIDDEN", "guild owner cannot leave; transfer ownership or delete the guild")
	}

	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return NotFound("NOT_FOUND", "you are not a member of this guild")
	}

	if err := s.members.Delete(ctx, guildID, userID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberRemove, map[string]any{"guild_id": guildID, "user_id": userID})
	return nil
}
