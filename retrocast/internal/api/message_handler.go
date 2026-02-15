package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// MessageHandler handles message CRUD endpoints.
type MessageHandler struct {
	messages   database.MessageRepository
	channels   database.ChannelRepository
	dmChannels database.DMChannelRepository
	members    database.MemberRepository
	roles      database.RoleRepository
	guilds     database.GuildRepository
	overrides  database.ChannelOverrideRepository
	snowflake  *snowflake.Generator
	gateway    gateway.Dispatcher
}

// NewMessageHandler creates a MessageHandler.
func NewMessageHandler(
	messages database.MessageRepository,
	channels database.ChannelRepository,
	dmChannels database.DMChannelRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	guilds database.GuildRepository,
	overrides database.ChannelOverrideRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
) *MessageHandler {
	return &MessageHandler{
		messages:   messages,
		channels:   channels,
		dmChannels: dmChannels,
		members:    members,
		roles:      roles,
		guilds:     guilds,
		overrides:  overrides,
		snowflake:  sf,
		gateway:    gw,
	}
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

// SendMessage handles POST /api/v1/channels/:id/messages.
func (h *MessageHandler) SendMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	// Try guild channel first, then DM channel.
	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var isDM bool
	if channel == nil && h.dmChannels != nil {
		dm, dmErr := h.dmChannels.GetByID(ctx, channelID)
		if dmErr != nil {
			return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if dm == nil {
			return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
		}
		ok, recipErr := h.dmChannels.IsRecipient(ctx, channelID, userID)
		if recipErr != nil {
			return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if !ok {
			return Error(c, http.StatusForbidden, "FORBIDDEN", "you are not a recipient of this DM")
		}
		isDM = true
	} else if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	if !isDM {
		if err := h.requirePermission(c, channel.GuildID, channelID, userID, permissions.PermSendMessages); err != nil {
			return err
		}
	}

	var req sendMessageRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if len(req.Content) == 0 || len(req.Content) > 2000 {
		return Error(c, http.StatusBadRequest, "INVALID_CONTENT", "message content must be 1-2000 characters")
	}

	msg := &models.Message{
		ID:        h.snowflake.Generate().Int64(),
		ChannelID: channelID,
		AuthorID:  userID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	if err := h.messages.Create(ctx, msg); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	full, err := h.messages.GetByID(ctx, msg.ID)
	if err != nil || full == nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if isDM {
		dm, _ := h.dmChannels.GetByID(ctx, channelID)
		if dm != nil {
			for _, r := range dm.Recipients {
				h.gateway.DispatchToUser(r.ID, gateway.EventMessageCreate, full)
			}
		}
	} else {
		h.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageCreate, full)
	}

	return c.JSON(http.StatusCreated, full)
}

// GetMessages handles GET /api/v1/channels/:id/messages.
func (h *MessageHandler) GetMessages(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if channel == nil && h.dmChannels != nil {
		if err := h.requireDMRecipient(c, channelID, userID); err != nil {
			return err
		}
	} else if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	} else {
		if err := h.requirePermission(c, channel.GuildID, channelID, userID, permissions.PermReadMessageHistory); err != nil {
			return err
		}
	}

	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed < 1 || parsed > 100 {
			return Error(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be 1-100")
		}
		limit = parsed
	}

	var before *int64
	if b := c.QueryParam("before"); b != "" {
		parsed, err := strconv.ParseInt(b, 10, 64)
		if err != nil {
			return Error(c, http.StatusBadRequest, "INVALID_BEFORE", "invalid before cursor")
		}
		before = &parsed
	}

	messages, err := h.messages.GetByChannelID(ctx, channelID, before, limit)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if messages == nil {
		messages = []models.MessageWithAuthor{}
	}
	return c.JSON(http.StatusOK, messages)
}

// GetMessage handles GET /api/v1/channels/:id/messages/:msg_id.
func (h *MessageHandler) GetMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if channel == nil && h.dmChannels != nil {
		if err := h.requireDMRecipient(c, channelID, userID); err != nil {
			return err
		}
	} else if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	} else {
		if err := h.requirePermission(c, channel.GuildID, channelID, userID, permissions.PermReadMessageHistory); err != nil {
			return err
		}
	}

	msg, err := h.messages.GetByID(ctx, msgID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "message not found")
	}

	return c.JSON(http.StatusOK, msg)
}

type editMessageRequest struct {
	Content string `json:"content"`
}

// EditMessage handles PATCH /api/v1/channels/:id/messages/:msg_id.
func (h *MessageHandler) EditMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var isDM bool
	if channel == nil && h.dmChannels != nil {
		if err := h.requireDMRecipient(c, channelID, userID); err != nil {
			return err
		}
		isDM = true
	} else if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	msg, err := h.messages.GetByID(ctx, msgID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "message not found")
	}

	if msg.AuthorID != userID {
		return Error(c, http.StatusForbidden, "FORBIDDEN", "you can only edit your own messages")
	}

	var req editMessageRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if len(req.Content) == 0 || len(req.Content) > 2000 {
		return Error(c, http.StatusBadRequest, "INVALID_CONTENT", "message content must be 1-2000 characters")
	}

	now := time.Now()
	updated := &models.Message{
		ID:       msgID,
		Content:  req.Content,
		EditedAt: &now,
	}

	if err := h.messages.Update(ctx, updated); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	full, err := h.messages.GetByID(ctx, msgID)
	if err != nil || full == nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if isDM {
		dm, _ := h.dmChannels.GetByID(ctx, channelID)
		if dm != nil {
			for _, r := range dm.Recipients {
				h.gateway.DispatchToUser(r.ID, gateway.EventMessageUpdate, full)
			}
		}
	} else {
		h.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageUpdate, full)
	}

	return c.JSON(http.StatusOK, full)
}

// DeleteMessage handles DELETE /api/v1/channels/:id/messages/:msg_id.
func (h *MessageHandler) DeleteMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var isDM bool
	if channel == nil && h.dmChannels != nil {
		if err := h.requireDMRecipient(c, channelID, userID); err != nil {
			return err
		}
		isDM = true
	} else if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	msg, err := h.messages.GetByID(ctx, msgID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if msg == nil || msg.ChannelID != channelID {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "message not found")
	}

	// In DMs, only the author can delete their own messages.
	// In guilds, author can delete their own; otherwise need MANAGE_MESSAGES.
	if msg.AuthorID != userID {
		if isDM {
			return Error(c, http.StatusForbidden, "FORBIDDEN", "you can only delete your own messages in DMs")
		}
		if err := h.requirePermission(c, channel.GuildID, channelID, userID, permissions.PermManageMessages); err != nil {
			return err
		}
	}

	if err := h.messages.Delete(ctx, msgID); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	deletePayload := struct {
		ID        int64 `json:"id,string"`
		ChannelID int64 `json:"channel_id,string"`
	}{ID: msgID, ChannelID: channelID}

	if isDM {
		dm, _ := h.dmChannels.GetByID(ctx, channelID)
		if dm != nil {
			for _, r := range dm.Recipients {
				h.gateway.DispatchToUser(r.ID, gateway.EventMessageDelete, deletePayload)
			}
		}
	} else {
		h.gateway.DispatchToGuild(channel.GuildID, gateway.EventMessageDelete, deletePayload)
	}

	return c.NoContent(http.StatusNoContent)
}

// Typing handles POST /api/v1/channels/:id/typing.
func (h *MessageHandler) Typing(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var isDM bool
	if channel == nil && h.dmChannels != nil {
		if err := h.requireDMRecipient(c, channelID, userID); err != nil {
			return err
		}
		isDM = true
	} else if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	if !isDM {
		if err := h.requirePermission(c, channel.GuildID, channelID, userID, permissions.PermSendMessages); err != nil {
			return err
		}
	}

	typingData := gateway.TypingStartData{
		ChannelID: channelID,
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	}

	if isDM {
		dm, _ := h.dmChannels.GetByID(ctx, channelID)
		if dm != nil {
			for _, r := range dm.Recipients {
				if r.ID != userID {
					h.gateway.DispatchToUser(r.ID, gateway.EventTypingStart, typingData)
				}
			}
		}
	} else {
		typingData.GuildID = channel.GuildID
		h.gateway.DispatchToGuild(channel.GuildID, gateway.EventTypingStart, typingData)
	}

	return c.NoContent(http.StatusNoContent)
}

// requireDMRecipient checks that the DM channel exists and the user is a recipient.
func (h *MessageHandler) requireDMRecipient(c echo.Context, channelID, userID int64) error {
	ctx := c.Request().Context()
	dm, err := h.dmChannels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if dm == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}
	ok, err := h.dmChannels.IsRecipient(ctx, channelID, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if !ok {
		return Error(c, http.StatusForbidden, "FORBIDDEN", "you are not a recipient of this DM")
	}
	return nil
}

// requirePermission checks that the user has the given permission in the channel,
// applying channel-level overrides on top of guild-level base permissions.
// Guild owners implicitly have all permissions.
func (h *MessageHandler) requirePermission(c echo.Context, guildID, channelID, userID int64, perm permissions.Permission) error {
	ctx := c.Request().Context()

	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == userID {
		return nil
	}

	member, err := h.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return Error(c, http.StatusForbidden, "FORBIDDEN", "you are not a member of this guild")
	}

	memberRoles, err := h.roles.GetByMember(ctx, guildID, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	allRoles, err := h.roles.GetByGuildID(ctx, guildID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var everyoneRole models.Role
	for _, r := range allRoles {
		if r.IsDefault {
			everyoneRole = r
			break
		}
	}

	basePerms := permissions.ComputeBasePermissions(everyoneRole, memberRoles)

	// Fetch channel overrides.
	channelOverrides, err := h.overrides.GetByChannel(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Separate @everyone override from role-specific overrides.
	var everyoneOverride *models.ChannelOverride
	var roleOverrides []models.ChannelOverride

	// Build set of member's role IDs for filtering.
	memberRoleIDs := make(map[int64]bool, len(memberRoles))
	for _, r := range memberRoles {
		memberRoleIDs[r.ID] = true
	}

	for i := range channelOverrides {
		if channelOverrides[i].RoleID == everyoneRole.ID {
			everyoneOverride = &channelOverrides[i]
		} else if memberRoleIDs[channelOverrides[i].RoleID] {
			roleOverrides = append(roleOverrides, channelOverrides[i])
		}
	}

	computed := permissions.ComputeChannelPermissions(basePerms, everyoneOverride, roleOverrides)
	if !computed.Has(perm) {
		return Error(c, http.StatusForbidden, "MISSING_PERMISSIONS", "you do not have the required permissions")
	}

	return nil
}
