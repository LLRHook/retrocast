package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// ChannelHandler handles channel CRUD endpoints.
type ChannelHandler struct {
	channels  database.ChannelRepository
	guilds    database.GuildRepository
	members   database.MemberRepository
	roles     database.RoleRepository
	snowflake *snowflake.Generator
	guildPerm func(ctx echo.Context, guildID, userID, perm int64) error
	gateway   gateway.Dispatcher
}

// NewChannelHandler creates a ChannelHandler. It takes the guild handler's
// requirePermission method for reuse.
func NewChannelHandler(
	channels database.ChannelRepository,
	guilds database.GuildRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	sf *snowflake.Generator,
	guildPerm func(ctx echo.Context, guildID, userID, perm int64) error,
	gw gateway.Dispatcher,
) *ChannelHandler {
	return &ChannelHandler{
		channels:  channels,
		guilds:    guilds,
		members:   members,
		roles:     roles,
		snowflake: sf,
		guildPerm: guildPerm,
		gateway:   gw,
	}
}

type createChannelRequest struct {
	Name     string          `json:"name"`
	Type     models.ChannelType `json:"type"`
	Topic    *string         `json:"topic"`
	ParentID *int64          `json:"parent_id,string"`
}

// CreateChannel handles POST /api/v1/guilds/:id/channels.
func (h *ChannelHandler) CreateChannel(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)
	if err := h.guildPerm(c, guildID, userID, PermManageChannels); err != nil {
		return err
	}

	var req createChannelRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}
	if len(req.Name) < 1 || len(req.Name) > 100 {
		return errorJSON(c, http.StatusBadRequest, "INVALID_NAME", "channel name must be 1-100 characters")
	}

	switch req.Type {
	case models.ChannelTypeText, models.ChannelTypeVoice, models.ChannelTypeCategory:
	default:
		return errorJSON(c, http.StatusBadRequest, "INVALID_TYPE", "channel type must be 0 (text), 2 (voice), or 4 (category)")
	}

	ctx := c.Request().Context()

	// Determine position by counting existing channels.
	existing, err := h.channels.GetByGuildID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	ch := &models.Channel{
		ID:       h.snowflake.Generate().Int64(),
		GuildID:  guildID,
		Name:     req.Name,
		Type:     req.Type,
		Position: len(existing),
		Topic:    req.Topic,
		ParentID: req.ParentID,
	}

	if err := h.channels.Create(ctx, ch); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusCreated, map[string]any{"data": ch})
}

// ListChannels handles GET /api/v1/guilds/:id/channels.
func (h *ChannelHandler) ListChannels(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	// Verify membership.
	member, err := h.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	channels, err := h.channels.GetByGuildID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if channels == nil {
		channels = []models.Channel{}
	}

	return c.JSON(http.StatusOK, map[string]any{"data": channels})
}

// GetChannel handles GET /api/v1/channels/:id.
func (h *ChannelHandler) GetChannel(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	ch, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if ch == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	// Verify membership in the channel's guild.
	member, err := h.members.GetByGuildAndUser(ctx, ch.GuildID, userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	return c.JSON(http.StatusOK, map[string]any{"data": ch})
}

type updateChannelRequest struct {
	Name     *string `json:"name"`
	Topic    *string `json:"topic"`
	Position *int    `json:"position"`
}

// UpdateChannel handles PATCH /api/v1/channels/:id.
func (h *ChannelHandler) UpdateChannel(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	ch, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if ch == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	if err := h.guildPerm(c, ch.GuildID, userID, PermManageChannels); err != nil {
		return err
	}

	var req updateChannelRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if req.Name != nil {
		if len(*req.Name) < 1 || len(*req.Name) > 100 {
			return errorJSON(c, http.StatusBadRequest, "INVALID_NAME", "channel name must be 1-100 characters")
		}
		ch.Name = *req.Name
	}
	if req.Topic != nil {
		ch.Topic = req.Topic
	}
	if req.Position != nil {
		ch.Position = *req.Position
	}

	if err := h.channels.Update(ctx, ch); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusOK, map[string]any{"data": ch})
}

// DeleteChannel handles DELETE /api/v1/channels/:id.
func (h *ChannelHandler) DeleteChannel(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	ch, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if ch == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	if err := h.guildPerm(c, ch.GuildID, userID, PermManageChannels); err != nil {
		return err
	}

	if err := h.channels.Delete(ctx, channelID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.NoContent(http.StatusNoContent)
}
