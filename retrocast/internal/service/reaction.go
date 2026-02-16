package service

import (
	"context"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// ReactionService handles reaction business logic.
type ReactionService struct {
	reactions  database.ReactionRepository
	messages   database.MessageRepository
	channels   database.ChannelRepository
	dmChannels database.DMChannelRepository
	gateway    gateway.Dispatcher
	perms      *PermissionChecker
}

// NewReactionService creates a ReactionService.
func NewReactionService(
	reactions database.ReactionRepository,
	messages database.MessageRepository,
	channels database.ChannelRepository,
	dmChannels database.DMChannelRepository,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *ReactionService {
	return &ReactionService{
		reactions:  reactions,
		messages:   messages,
		channels:   channels,
		dmChannels: dmChannels,
		gateway:    gw,
		perms:      perms,
	}
}

// reactionEventData is the gateway payload for reaction add/remove events.
type reactionEventData struct {
	MessageID int64  `json:"message_id,string"`
	ChannelID int64  `json:"channel_id,string"`
	GuildID   int64  `json:"guild_id,string,omitempty"`
	UserID    int64  `json:"user_id,string"`
	Emoji     string `json:"emoji"`
}

// AddReaction adds a reaction to a message.
func (s *ReactionService) AddReaction(ctx context.Context, channelID, messageID, userID int64, emoji string) error {
	if emoji == "" {
		return BadRequest("INVALID_EMOJI", "emoji must not be empty")
	}

	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermViewChannel|permissions.PermReadMessageHistory); err != nil {
			return err
		}
	}

	msg, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return NotFound("NOT_FOUND", "message not found")
	}

	if err := s.reactions.Add(ctx, messageID, userID, emoji); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	event := reactionEventData{
		MessageID: messageID,
		ChannelID: channelID,
		UserID:    userID,
		Emoji:     emoji,
	}

	if isDM {
		s.dispatchToDM(ctx, channelID, gateway.EventMessageReactionAdd, event)
	} else {
		event.GuildID = channel.GuildID
		s.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageReactionAdd, event)
	}

	return nil
}

// RemoveReaction removes a reaction from a message.
func (s *ReactionService) RemoveReaction(ctx context.Context, channelID, messageID, userID int64, emoji string) error {
	if emoji == "" {
		return BadRequest("INVALID_EMOJI", "emoji must not be empty")
	}

	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermViewChannel|permissions.PermReadMessageHistory); err != nil {
			return err
		}
	}

	if err := s.reactions.Remove(ctx, messageID, userID, emoji); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	event := reactionEventData{
		MessageID: messageID,
		ChannelID: channelID,
		UserID:    userID,
		Emoji:     emoji,
	}

	if isDM {
		s.dispatchToDM(ctx, channelID, gateway.EventMessageReactionRemove, event)
	} else {
		event.GuildID = channel.GuildID
		s.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageReactionRemove, event)
	}

	return nil
}

// GetReactions returns the user IDs who reacted with a specific emoji on a message.
func (s *ReactionService) GetReactions(ctx context.Context, channelID, messageID, userID int64, emoji string, limit int) ([]int64, error) {
	if emoji == "" {
		return nil, BadRequest("INVALID_EMOJI", "emoji must not be empty")
	}

	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermViewChannel|permissions.PermReadMessageHistory); err != nil {
			return nil, err
		}
	}

	userIDs, err := s.reactions.GetUsersByReaction(ctx, messageID, emoji, limit)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if userIDs == nil {
		userIDs = []int64{}
	}
	return userIDs, nil
}

// resolveChannelAccess returns the guild channel (if any) and whether this is a DM.
func (s *ReactionService) resolveChannelAccess(ctx context.Context, channelID, userID int64) (*models.Channel, bool, error) {
	channel, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, false, Internal("INTERNAL", "internal server error")
	}

	if channel == nil && s.dmChannels != nil {
		dm, dmErr := s.dmChannels.GetByID(ctx, channelID)
		if dmErr != nil {
			return nil, false, Internal("INTERNAL", "internal server error")
		}
		if dm == nil {
			return nil, false, NotFound("NOT_FOUND", "channel not found")
		}
		ok, recipErr := s.dmChannels.IsRecipient(ctx, channelID, userID)
		if recipErr != nil {
			return nil, false, Internal("INTERNAL", "internal server error")
		}
		if !ok {
			return nil, false, Forbidden("FORBIDDEN", "you are not a recipient of this DM")
		}
		return nil, true, nil
	} else if channel == nil {
		return nil, false, NotFound("NOT_FOUND", "channel not found")
	}

	return channel, false, nil
}

// dispatchToDM dispatches an event to all DM recipients.
func (s *ReactionService) dispatchToDM(ctx context.Context, channelID int64, event string, data any) {
	dm, _ := s.dmChannels.GetByID(ctx, channelID)
	if dm != nil {
		for _, r := range dm.Recipients {
			s.gateway.DispatchToUser(r.ID, event, data)
		}
	}
}
