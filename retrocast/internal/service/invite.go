package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// InviteInfo is the public-facing invite information (no auth required).
type InviteInfo struct {
	Code        string `json:"code"`
	GuildName   string `json:"guild_name"`
	MemberCount int    `json:"member_count"`
	CreatorID   int64  `json:"creator_id,string"`
}

// InviteService handles invite business logic.
type InviteService struct {
	invites database.InviteRepository
	guilds  database.GuildRepository
	members database.MemberRepository
	bans    database.BanRepository
	gateway gateway.Dispatcher
	perms   *PermissionChecker
}

// NewInviteService creates an InviteService.
func NewInviteService(
	invites database.InviteRepository,
	guilds database.GuildRepository,
	members database.MemberRepository,
	bans database.BanRepository,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *InviteService {
	return &InviteService{
		invites: invites,
		guilds:  guilds,
		members: members,
		bans:    bans,
		gateway: gw,
		perms:   perms,
	}
}

// CreateInvite creates an invite for a guild.
func (s *InviteService) CreateInvite(ctx context.Context, guildID, userID int64, maxUses, maxAgeSeconds int) (*models.Invite, error) {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, userID, permissions.PermCreateInvite); err != nil {
		return nil, err
	}

	if maxAgeSeconds == 0 {
		maxAgeSeconds = 86400
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	now := time.Now()
	var expiresAt *time.Time
	if maxAgeSeconds > 0 {
		t := now.Add(time.Duration(maxAgeSeconds) * time.Second)
		expiresAt = &t
	}

	invite := &models.Invite{
		Code:      code,
		GuildID:   guildID,
		CreatorID: userID,
		MaxUses:   maxUses,
		Uses:      0,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	if err := s.invites.Create(ctx, invite); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return invite, nil
}

// ListInvites returns all invites for a guild.
func (s *InviteService) ListInvites(ctx context.Context, guildID, userID int64) ([]models.Invite, error) {
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, userID, permissions.PermManageGuild); err != nil {
		return nil, err
	}

	invites, err := s.invites.GetByGuildID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if invites == nil {
		invites = []models.Invite{}
	}
	return invites, nil
}

// GetInvite returns public invite information (no auth).
func (s *InviteService) GetInvite(ctx context.Context, code string) (*InviteInfo, error) {
	invite, err := s.invites.GetByCode(ctx, code)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if invite == nil {
		return nil, NotFound("NOT_FOUND", "invite not found")
	}

	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, NotFound("EXPIRED", "invite has expired")
	}

	guild, err := s.guilds.GetByID(ctx, invite.GuildID)
	if err != nil || guild == nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	members, err := s.members.GetByGuildID(ctx, invite.GuildID, 10000, 0)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return &InviteInfo{
		Code:        invite.Code,
		GuildName:   guild.Name,
		MemberCount: len(members),
		CreatorID:   invite.CreatorID,
	}, nil
}

// AcceptInvite joins the user to the guild via invite.
func (s *InviteService) AcceptInvite(ctx context.Context, code string, userID int64) (*models.Guild, error) {
	invite, err := s.invites.GetByCode(ctx, code)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if invite == nil {
		return nil, NotFound("NOT_FOUND", "invite not found")
	}

	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, Gone("EXPIRED", "invite has expired")
	}

	if invite.MaxUses > 0 && invite.Uses >= invite.MaxUses {
		return nil, Gone("MAX_USES", "invite has reached maximum uses")
	}

	existing, err := s.members.GetByGuildAndUser(ctx, invite.GuildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if existing != nil {
		return nil, Conflict("ALREADY_MEMBER", "you are already a member of this guild")
	}

	ban, err := s.bans.GetByGuildAndUser(ctx, invite.GuildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if ban != nil {
		return nil, Forbidden("BANNED", "you are banned from this guild")
	}

	member := &models.Member{
		GuildID:  invite.GuildID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	if err := s.members.Create(ctx, member); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if err := s.invites.IncrementUses(ctx, code); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	guild, err := s.guilds.GetByID(ctx, invite.GuildID)
	if err != nil || guild == nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(invite.GuildID, gateway.EventGuildMemberAdd, member)

	return guild, nil
}

// RevokeInvite deletes an invite. The creator can always revoke; otherwise need MANAGE_GUILD.
func (s *InviteService) RevokeInvite(ctx context.Context, code string, userID int64) error {
	invite, err := s.invites.GetByCode(ctx, code)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if invite == nil {
		return NotFound("NOT_FOUND", "invite not found")
	}

	if invite.CreatorID != userID {
		if err := s.perms.RequireGuildPermissionByPerm(ctx, invite.GuildID, userID, permissions.PermManageGuild); err != nil {
			return err
		}
	}

	if err := s.invites.Delete(ctx, code); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	return nil
}

func generateInviteCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
