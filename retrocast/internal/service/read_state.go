package service

import (
	"context"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// ReadStateService handles read state business logic.
type ReadStateService struct {
	readStates database.ReadStateRepository
	channels   database.ChannelRepository
	dmChannels database.DMChannelRepository
	perms      *PermissionChecker
}

// NewReadStateService creates a ReadStateService.
func NewReadStateService(
	readStates database.ReadStateRepository,
	channels database.ChannelRepository,
	dmChannels database.DMChannelRepository,
	perms *PermissionChecker,
) *ReadStateService {
	return &ReadStateService{
		readStates: readStates,
		channels:   channels,
		dmChannels: dmChannels,
		perms:      perms,
	}
}

// Ack marks a channel as read up to the given message ID.
func (s *ReadStateService) Ack(ctx context.Context, channelID, messageID, userID int64) error {
	// Verify channel access
	channel, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	if channel == nil && s.dmChannels != nil {
		dm, dmErr := s.dmChannels.GetByID(ctx, channelID)
		if dmErr != nil {
			return Internal("INTERNAL", "internal server error")
		}
		if dm == nil {
			return NotFound("NOT_FOUND", "channel not found")
		}
		ok, recipErr := s.dmChannels.IsRecipient(ctx, channelID, userID)
		if recipErr != nil {
			return Internal("INTERNAL", "internal server error")
		}
		if !ok {
			return Forbidden("FORBIDDEN", "you are not a recipient of this DM")
		}
	} else if channel == nil {
		return NotFound("NOT_FOUND", "channel not found")
	} else {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermViewChannel); err != nil {
			return err
		}
	}

	if err := s.readStates.Upsert(ctx, userID, channelID, messageID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	return nil
}

// GetReadStates returns all read states for a user.
func (s *ReadStateService) GetReadStates(ctx context.Context, userID int64) ([]models.ReadState, error) {
	states, err := s.readStates.GetByUser(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if states == nil {
		states = []models.ReadState{}
	}
	return states, nil
}
