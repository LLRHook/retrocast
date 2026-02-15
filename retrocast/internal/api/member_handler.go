package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// MemberHandler handles member endpoints.
type MemberHandler struct {
	service *service.MemberService
}

// NewMemberHandler creates a MemberHandler.
func NewMemberHandler(svc *service.MemberService) *MemberHandler {
	return &MemberHandler{service: svc}
}

// ListMembers handles GET /api/v1/guilds/:id/members.
func (h *MemberHandler) ListMembers(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	members, err := h.service.ListMembers(c.Request().Context(), guildID, userID, limit, offset)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": members})
}

// GetMember handles GET /api/v1/guilds/:id/members/:user_id.
func (h *MemberHandler) GetMember(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
	}

	callerID := auth.GetUserID(c)

	member, err := h.service.GetMember(c.Request().Context(), guildID, callerID, targetUserID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": member})
}

type updateMemberRequest struct {
	Nickname *string  `json:"nickname"`
	Roles    *[]int64 `json:"roles"`
}

// UpdateSelf handles PATCH /api/v1/guilds/:id/members/@me.
func (h *MemberHandler) UpdateSelf(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	var req updateMemberRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	member, err := h.service.UpdateSelf(c.Request().Context(), guildID, userID, req.Nickname)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": member})
}

// UpdateMember handles PATCH /api/v1/guilds/:id/members/:user_id.
func (h *MemberHandler) UpdateMember(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
	}

	callerID := auth.GetUserID(c)

	var req updateMemberRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	member, err := h.service.UpdateMember(c.Request().Context(), guildID, callerID, targetUserID, req.Nickname, req.Roles)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": member})
}

// KickMember handles DELETE /api/v1/guilds/:id/members/:user_id.
func (h *MemberHandler) KickMember(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
	}

	callerID := auth.GetUserID(c)

	if err := h.service.KickMember(c.Request().Context(), guildID, callerID, targetUserID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// LeaveGuild handles DELETE /api/v1/guilds/:id/members/@me.
func (h *MemberHandler) LeaveGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.LeaveGuild(c.Request().Context(), guildID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
