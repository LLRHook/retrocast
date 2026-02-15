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

// MessageService handles message business logic for both guild and DM channels.
type MessageService struct {
	messages   database.MessageRepository
	channels   database.ChannelRepository
	dmChannels database.DMChannelRepository
	snowflake  *snowflake.Generator
	gateway    gateway.Dispatcher
	perms      *PermissionChecker
}

// NewMessageService creates a MessageService.
func NewMessageService(
	messages database.MessageRepository,
	channels database.ChannelRepository,
	dmChannels database.DMChannelRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
) *MessageService {
	return &MessageService{
		messages:   messages,
		channels:   channels,
		dmChannels: dmChannels,
		snowflake:  sf,
		gateway:    gw,
		perms:      perms,
	}
}

// SendMessage creates a message in a guild or DM channel.
func (s *MessageService) SendMessage(ctx context.Context, channelID, userID int64, content string) (*models.MessageWithAuthor, error) {
	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermSendMessages); err != nil {
			return nil, err
		}
	}

	if len(content) == 0 || len(content) > 2000 {
		return nil, BadRequest("INVALID_CONTENT", "message content must be 1-2000 characters")
	}

	msg := &models.Message{
		ID:        s.snowflake.Generate().Int64(),
		ChannelID: channelID,
		AuthorID:  userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	full, err := s.messages.GetByID(ctx, msg.ID)
	if err != nil || full == nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if isDM {
		s.dispatchToDM(ctx, channelID, gateway.EventMessageCreate, full)
	} else {
		s.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageCreate, full)
	}

	return full, nil
}

// GetMessages returns messages from a channel with cursor-based pagination.
func (s *MessageService) GetMessages(ctx context.Context, channelID, userID int64, before *int64, limit int) ([]models.MessageWithAuthor, error) {
	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermReadMessageHistory); err != nil {
			return nil, err
		}
	}

	messages, err := s.messages.GetByChannelID(ctx, channelID, before, limit)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if messages == nil {
		messages = []models.MessageWithAuthor{}
	}
	return messages, nil
}

// GetMessage returns a single message by ID.
func (s *MessageService) GetMessage(ctx context.Context, channelID, msgID, userID int64) (*models.MessageWithAuthor, error) {
	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermReadMessageHistory); err != nil {
			return nil, err
		}
	}

	msg, err := s.messages.GetByID(ctx, msgID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return nil, NotFound("NOT_FOUND", "message not found")
	}

	return msg, nil
}

// EditMessage edits a message. Only the author can edit.
func (s *MessageService) EditMessage(ctx context.Context, channelID, msgID, userID int64, content string) (*models.MessageWithAuthor, error) {
	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	msg, err := s.messages.GetByID(ctx, msgID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return nil, NotFound("NOT_FOUND", "message not found")
	}

	if msg.AuthorID != userID {
		return nil, Forbidden("FORBIDDEN", "you can only edit your own messages")
	}

	if len(content) == 0 || len(content) > 2000 {
		return nil, BadRequest("INVALID_CONTENT", "message content must be 1-2000 characters")
	}

	now := time.Now()
	updated := &models.Message{
		ID:       msgID,
		Content:  content,
		EditedAt: &now,
	}

	if err := s.messages.Update(ctx, updated); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	full, err := s.messages.GetByID(ctx, msgID)
	if err != nil || full == nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if isDM {
		s.dispatchToDM(ctx, channelID, gateway.EventMessageUpdate, full)
	} else {
		s.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageUpdate, full)
	}

	return full, nil
}

// DeleteMessage deletes a message. Author can always delete their own;
// in guilds, MANAGE_MESSAGES permission allows deleting others' messages.
func (s *MessageService) DeleteMessage(ctx context.Context, channelID, msgID, userID int64) error {
	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return err
	}

	msg, err := s.messages.GetByID(ctx, msgID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return NotFound("NOT_FOUND", "message not found")
	}

	if msg.AuthorID != userID {
		if isDM {
			return Forbidden("FORBIDDEN", "you can only delete your own messages in DMs")
		}
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermManageMessages); err != nil {
			return err
		}
	}

	if err := s.messages.Delete(ctx, msgID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	deletePayload := struct {
		ID        int64 `json:"id,string"`
		ChannelID int64 `json:"channel_id,string"`
	}{ID: msgID, ChannelID: channelID}

	if isDM {
		s.dispatchToDM(ctx, channelID, gateway.EventMessageDelete, deletePayload)
	} else {
		s.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageDelete, deletePayload)
	}

	return nil
}

// Typing dispatches a typing indicator event.
func (s *MessageService) Typing(ctx context.Context, channelID, userID int64) error {
	channel, isDM, err := s.resolveChannelAccess(ctx, channelID, userID)
	if err != nil {
		return err
	}

	if !isDM {
		if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermSendMessages); err != nil {
			return err
		}
	}

	typingData := gateway.TypingStartData{
		ChannelID: channelID,
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	}

	if isDM {
		dm, _ := s.dmChannels.GetByID(ctx, channelID)
		if dm != nil {
			for _, r := range dm.Recipients {
				if r.ID != userID {
					s.gateway.DispatchToUser(r.ID, gateway.EventTypingStart, typingData)
				}
			}
		}
	} else {
		typingData.GuildID = channel.GuildID
		s.gateway.DispatchToGuild(channel.GuildID, gateway.EventTypingStart, typingData)
	}

	return nil
}

// resolveChannelAccess returns the guild channel (if any) and whether this is a DM.
// It also verifies the user has access to the channel (membership or DM recipient).
func (s *MessageService) resolveChannelAccess(ctx context.Context, channelID, userID int64) (*models.Channel, bool, error) {
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
func (s *MessageService) dispatchToDM(ctx context.Context, channelID int64, event string, data any) {
	dm, _ := s.dmChannels.GetByID(ctx, channelID)
	if dm != nil {
		for _, r := range dm.Recipients {
			s.gateway.DispatchToUser(r.ID, event, data)
		}
	}
}
