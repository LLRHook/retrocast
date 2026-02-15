package service

import (
	"context"
	"time"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// GuildService handles guild business logic.
type GuildService struct {
	guilds    database.GuildRepository
	channels  database.ChannelRepository
	members   database.MemberRepository
	roles     database.RoleRepository
	snowflake *snowflake.Generator
	gateway   gateway.Dispatcher
	perms     *PermissionChecker
}

// NewGuildService creates a GuildService.
func NewGuildService(
	guilds database.GuildRepository,
	channels database.ChannelRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *GuildService {
	return &GuildService{
		guilds:    guilds,
		channels:  channels,
		members:   members,
		roles:     roles,
		snowflake: sf,
		gateway:   gw,
		perms:     perms,
	}
}

// CreateGuild creates a guild with default roles and channels.
func (s *GuildService) CreateGuild(ctx context.Context, userID int64, name string) (*models.Guild, error) {
	if len(name) < 2 || len(name) > 100 {
		return nil, BadRequest("INVALID_NAME", "guild name must be 2-100 characters")
	}

	now := time.Now()

	guild := &models.Guild{
		ID:        s.snowflake.Generate().Int64(),
		Name:      name,
		OwnerID:   userID,
		CreatedAt: now,
	}
	if err := s.guilds.Create(ctx, guild); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	everyoneRole := &models.Role{
		ID:          s.snowflake.Generate().Int64(),
		GuildID:     guild.ID,
		Name:        "@everyone",
		Permissions: int64(permissions.PermViewChannel | permissions.PermSendMessages | permissions.PermReadMessageHistory),
		Position:    0,
		IsDefault:   true,
	}
	if err := s.roles.Create(ctx, everyoneRole); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	adminRole := &models.Role{
		ID:          s.snowflake.Generate().Int64(),
		GuildID:     guild.ID,
		Name:        "Admin",
		Permissions: int64(permissions.PermAdministrator),
		Position:    1,
	}
	if err := s.roles.Create(ctx, adminRole); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	member := &models.Member{
		GuildID:  guild.ID,
		UserID:   userID,
		JoinedAt: now,
	}
	if err := s.members.Create(ctx, member); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if err := s.members.AddRole(ctx, guild.ID, userID, everyoneRole.ID); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if err := s.members.AddRole(ctx, guild.ID, userID, adminRole.ID); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	generalText := &models.Channel{
		ID:       s.snowflake.Generate().Int64(),
		GuildID:  guild.ID,
		Name:     "general",
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := s.channels.Create(ctx, generalText); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	generalVoice := &models.Channel{
		ID:       s.snowflake.Generate().Int64(),
		GuildID:  guild.ID,
		Name:     "General",
		Type:     models.ChannelTypeVoice,
		Position: 1,
	}
	if err := s.channels.Create(ctx, generalVoice); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToUser(userID, gateway.EventGuildCreate, guild)
	return guild, nil
}

// GetGuild returns a guild if the user is a member.
func (s *GuildService) GetGuild(ctx context.Context, guildID, userID int64) (*models.Guild, error) {
	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	guild, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	return guild, nil
}

// UpdateGuild updates guild name and/or icon.
func (s *GuildService) UpdateGuild(ctx context.Context, guildID, userID int64, name *string, icon *string) (*models.Guild, error) {
	if err := s.perms.RequireGuildPermission(ctx, guildID, userID, int64(permissions.PermManageGuild)); err != nil {
		return nil, err
	}

	guild, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	if name != nil {
		if len(*name) < 2 || len(*name) > 100 {
			return nil, BadRequest("INVALID_NAME", "guild name must be 2-100 characters")
		}
		guild.Name = *name
	}
	if icon != nil {
		guild.IconHash = icon
	}

	if err := s.guilds.Update(ctx, guild); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildUpdate, guild)
	return guild, nil
}

// DeleteGuild deletes a guild. Only the owner can delete.
func (s *GuildService) DeleteGuild(ctx context.Context, guildID, userID int64) error {
	guild, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if guild == nil {
		return NotFound("NOT_FOUND", "guild not found")
	}
	if guild.OwnerID != userID {
		return Forbidden("FORBIDDEN", "only the guild owner can delete the guild")
	}

	if err := s.guilds.Delete(ctx, guildID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventGuildDelete, map[string]any{"id": guildID})
	return nil
}

// ListMyGuilds returns all guilds the user is a member of.
func (s *GuildService) ListMyGuilds(ctx context.Context, userID int64) ([]models.Guild, error) {
	guilds, err := s.guilds.GetByUserID(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if guilds == nil {
		guilds = []models.Guild{}
	}
	return guilds, nil
}
