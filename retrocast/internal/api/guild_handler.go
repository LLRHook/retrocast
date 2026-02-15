package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// GuildHandler handles guild CRUD endpoints.
type GuildHandler struct {
	service *service.GuildService
}

// NewGuildHandler creates a GuildHandler.
func NewGuildHandler(svc *service.GuildService) *GuildHandler {
	return &GuildHandler{service: svc}
}

type createGuildRequest struct {
	Name string `json:"name"`
}

// CreateGuild handles POST /api/v1/guilds.
func (h *GuildHandler) CreateGuild(c echo.Context) error {
	var req createGuildRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	userID := auth.GetUserID(c)

	guild, err := h.service.CreateGuild(c.Request().Context(), userID, req.Name)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{"data": guild})
}

// GetGuild handles GET /api/v1/guilds/:id.
func (h *GuildHandler) GetGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	guild, err := h.service.GetGuild(c.Request().Context(), guildID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": guild})
}

type updateGuildRequest struct {
	Name *string `json:"name"`
	Icon *string `json:"icon"`
}

// UpdateGuild handles PATCH /api/v1/guilds/:id.
func (h *GuildHandler) UpdateGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	var req updateGuildRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	guild, err := h.service.UpdateGuild(c.Request().Context(), guildID, userID, req.Name, req.Icon)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": guild})
}

// DeleteGuild handles DELETE /api/v1/guilds/:id.
func (h *GuildHandler) DeleteGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.DeleteGuild(c.Request().Context(), guildID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ListMyGuilds handles GET /api/v1/users/@me/guilds.
func (h *GuildHandler) ListMyGuilds(c echo.Context) error {
	userID := auth.GetUserID(c)

	guilds, err := h.service.ListMyGuilds(c.Request().Context(), userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": guilds})
}
