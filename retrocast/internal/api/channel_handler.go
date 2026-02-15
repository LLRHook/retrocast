package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
)

// ChannelHandler handles channel CRUD endpoints.
type ChannelHandler struct {
	service *service.ChannelService
}

// NewChannelHandler creates a ChannelHandler.
func NewChannelHandler(svc *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{service: svc}
}

type createChannelRequest struct {
	Name     string             `json:"name"`
	Type     models.ChannelType `json:"type"`
	Topic    *string            `json:"topic"`
	ParentID *int64             `json:"parent_id,string"`
}

// CreateChannel handles POST /api/v1/guilds/:id/channels.
func (h *ChannelHandler) CreateChannel(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	var req createChannelRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	ch, err := h.service.CreateChannel(c.Request().Context(), guildID, userID, req.Name, req.Type, req.Topic, req.ParentID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{"data": ch})
}

// ListChannels handles GET /api/v1/guilds/:id/channels.
func (h *ChannelHandler) ListChannels(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	channels, err := h.service.ListChannels(c.Request().Context(), guildID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": channels})
}

// GetChannel handles GET /api/v1/channels/:id.
func (h *ChannelHandler) GetChannel(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	ch, err := h.service.GetChannel(c.Request().Context(), channelID, userID)
	if err != nil {
		return mapServiceError(c, err)
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

	userID := auth.GetUserID(c)

	var req updateChannelRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	ch, err := h.service.UpdateChannel(c.Request().Context(), channelID, userID, req.Name, req.Topic, req.Position)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": ch})
}

// DeleteChannel handles DELETE /api/v1/channels/:id.
func (h *ChannelHandler) DeleteChannel(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.DeleteChannel(c.Request().Context(), channelID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
