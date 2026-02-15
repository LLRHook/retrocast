package service

import (
	"context"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// ChannelService handles channel business logic.
type ChannelService struct {
	channels  database.ChannelRepository
	members   database.MemberRepository
	snowflake *snowflake.Generator
	gateway   gateway.Dispatcher
	perms     *PermissionChecker
}

// NewChannelService creates a ChannelService.
func NewChannelService(
	channels database.ChannelRepository,
	members database.MemberRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *ChannelService {
	return &ChannelService{
		channels:  channels,
		members:   members,
		snowflake: sf,
		gateway:   gw,
		perms:     perms,
	}
}

// CreateChannel creates a channel in the given guild.
func (s *ChannelService) CreateChannel(ctx context.Context, guildID, userID int64, name string, chType models.ChannelType, topic *string, parentID *int64) (*models.Channel, error) {
	if err := s.perms.RequireGuildPermission(ctx, guildID, userID, int64(permissions.PermManageChannels)); err != nil {
		return nil, err
	}

	if len(name) < 1 || len(name) > 100 {
		return nil, BadRequest("INVALID_NAME", "channel name must be 1-100 characters")
	}

	switch chType {
	case models.ChannelTypeText, models.ChannelTypeVoice, models.ChannelTypeCategory:
	default:
		return nil, BadRequest("INVALID_TYPE", "channel type must be 0 (text), 2 (voice), or 4 (category)")
	}

	existing, err := s.channels.GetByGuildID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	ch := &models.Channel{
		ID:       s.snowflake.Generate().Int64(),
		GuildID:  guildID,
		Name:     name,
		Type:     chType,
		Position: len(existing),
		Topic:    topic,
		ParentID: parentID,
	}

	if err := s.channels.Create(ctx, ch); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(guildID, gateway.EventChannelCreate, ch)
	return ch, nil
}

// ListChannels returns all channels in a guild if the user is a member.
func (s *ChannelService) ListChannels(ctx context.Context, guildID, userID int64) ([]models.Channel, error) {
	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "guild not found")
	}

	channels, err := s.channels.GetByGuildID(ctx, guildID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if channels == nil {
		channels = []models.Channel{}
	}

	return channels, nil
}

// GetChannel returns a channel if the user is a member of its guild.
func (s *ChannelService) GetChannel(ctx context.Context, channelID, userID int64) (*models.Channel, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if ch == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	member, err := s.members.GetByGuildAndUser(ctx, ch.GuildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	return ch, nil
}

// UpdateChannel updates channel name, topic, and/or position.
func (s *ChannelService) UpdateChannel(ctx context.Context, channelID, userID int64, name *string, topic *string, position *int) (*models.Channel, error) {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if ch == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	if err := s.perms.RequireGuildPermission(ctx, ch.GuildID, userID, int64(permissions.PermManageChannels)); err != nil {
		return nil, err
	}

	if name != nil {
		if len(*name) < 1 || len(*name) > 100 {
			return nil, BadRequest("INVALID_NAME", "channel name must be 1-100 characters")
		}
		ch.Name = *name
	}
	if topic != nil {
		ch.Topic = topic
	}
	if position != nil {
		ch.Position = *position
	}

	if err := s.channels.Update(ctx, ch); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(ch.GuildID, gateway.EventChannelUpdate, ch)
	return ch, nil
}

// DeleteChannel deletes a channel.
func (s *ChannelService) DeleteChannel(ctx context.Context, channelID, userID int64) error {
	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if ch == nil {
		return NotFound("NOT_FOUND", "channel not found")
	}

	if err := s.perms.RequireGuildPermission(ctx, ch.GuildID, userID, int64(permissions.PermManageChannels)); err != nil {
		return err
	}

	if err := s.channels.Delete(ctx, channelID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(ch.GuildID, gateway.EventChannelDelete, map[string]any{"id": channelID, "guild_id": ch.GuildID})
	return nil
}
