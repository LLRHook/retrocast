package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// BanHandler handles guild ban endpoints.
type BanHandler struct {
	service *service.BanService
}

// NewBanHandler creates a BanHandler.
func NewBanHandler(svc *service.BanService) *BanHandler {
	return &BanHandler{service: svc}
}

type banMemberRequest struct {
	Reason *string `json:"reason"`
}

// BanMember handles PUT /api/v1/guilds/:id/bans/:user_id.
func (h *BanHandler) BanMember(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
	}

	userID := auth.GetUserID(c)

	var req banMemberRequest
	_ = c.Bind(&req) // optional body

	if err := h.service.BanMember(c.Request().Context(), guildID, userID, targetUserID, req.Reason); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// UnbanMember handles DELETE /api/v1/guilds/:id/bans/:user_id.
func (h *BanHandler) UnbanMember(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.UnbanMember(c.Request().Context(), guildID, userID, targetUserID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ListBans handles GET /api/v1/guilds/:id/bans.
func (h *BanHandler) ListBans(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	bans, err := h.service.ListBans(c.Request().Context(), guildID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, bans)
}
