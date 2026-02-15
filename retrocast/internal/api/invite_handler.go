package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// InviteHandler handles invite endpoints.
type InviteHandler struct {
	service *service.InviteService
}

// NewInviteHandler creates an InviteHandler.
func NewInviteHandler(svc *service.InviteService) *InviteHandler {
	return &InviteHandler{service: svc}
}

type createInviteRequest struct {
	MaxUses       int `json:"max_uses"`
	MaxAgeSeconds int `json:"max_age_seconds"`
}

// CreateInvite handles POST /api/v1/guilds/:id/invites.
func (h *InviteHandler) CreateInvite(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	var req createInviteRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	invite, err := h.service.CreateInvite(c.Request().Context(), guildID, userID, req.MaxUses, req.MaxAgeSeconds)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, invite)
}

// ListInvites handles GET /api/v1/guilds/:id/invites.
func (h *InviteHandler) ListInvites(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	invites, err := h.service.ListInvites(c.Request().Context(), guildID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, invites)
}

// GetInvite handles GET /api/v1/invites/:code (no auth required).
func (h *InviteHandler) GetInvite(c echo.Context) error {
	code := c.Param("code")

	info, err := h.service.GetInvite(c.Request().Context(), code)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, info)
}

// AcceptInvite handles POST /api/v1/invites/:code (auth required).
func (h *InviteHandler) AcceptInvite(c echo.Context) error {
	code := c.Param("code")
	userID := auth.GetUserID(c)

	guild, err := h.service.AcceptInvite(c.Request().Context(), code, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, guild)
}

// RevokeInvite handles DELETE /api/v1/invites/:code.
func (h *InviteHandler) RevokeInvite(c echo.Context) error {
	code := c.Param("code")
	userID := auth.GetUserID(c)

	if err := h.service.RevokeInvite(c.Request().Context(), code, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
