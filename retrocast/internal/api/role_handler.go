package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// RoleHandler handles role and channel override endpoints.
type RoleHandler struct {
	service *service.RoleService
}

// NewRoleHandler creates a RoleHandler.
func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{service: svc}
}

type createRoleRequest struct {
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Permissions int64  `json:"permissions,string"`
	Position    int    `json:"position"`
}

// CreateRole handles POST /api/v1/guilds/:id/roles.
func (h *RoleHandler) CreateRole(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	var req createRoleRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	actorID := auth.GetUserID(c)

	role, err := h.service.CreateRole(c.Request().Context(), guildID, actorID, req.Name, req.Color, req.Permissions, req.Position)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, role)
}

// ListRoles handles GET /api/v1/guilds/:id/roles.
func (h *RoleHandler) ListRoles(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	roles, err := h.service.ListRoles(c.Request().Context(), guildID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, roles)
}

type updateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Color       *int    `json:"color,omitempty"`
	Permissions *int64  `json:"permissions,string,omitempty"`
	Position    *int    `json:"position,omitempty"`
}

// UpdateRole handles PATCH /api/v1/guilds/:id/roles/:role_id.
func (h *RoleHandler) UpdateRole(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid role id")
	}

	var req updateRoleRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	actorID := auth.GetUserID(c)

	role, err := h.service.UpdateRole(c.Request().Context(), guildID, actorID, roleID, req.Name, req.Color, req.Permissions, req.Position)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, role)
}

// DeleteRole handles DELETE /api/v1/guilds/:id/roles/:role_id.
func (h *RoleHandler) DeleteRole(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid role id")
	}

	actorID := auth.GetUserID(c)

	if err := h.service.DeleteRole(c.Request().Context(), guildID, actorID, roleID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// AssignRole handles PUT /api/v1/guilds/:id/members/:user_id/roles/:role_id.
func (h *RoleHandler) AssignRole(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid user id")
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid role id")
	}

	actorID := auth.GetUserID(c)

	if err := h.service.AssignRole(c.Request().Context(), guildID, actorID, userID, roleID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveRole handles DELETE /api/v1/guilds/:id/members/:user_id/roles/:role_id.
func (h *RoleHandler) RemoveRole(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid user id")
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid role id")
	}

	actorID := auth.GetUserID(c)

	if err := h.service.RemoveRole(c.Request().Context(), guildID, actorID, userID, roleID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

type setOverrideRequest struct {
	Allow int64 `json:"allow,string"`
	Deny  int64 `json:"deny,string"`
}

// SetChannelOverride handles PUT /api/v1/channels/:id/permissions/:role_id.
func (h *RoleHandler) SetChannelOverride(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel id")
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid role id")
	}

	var req setOverrideRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	override, err := h.service.SetChannelOverride(c.Request().Context(), channelID, roleID, req.Allow, req.Deny)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, override)
}

// DeleteChannelOverride handles DELETE /api/v1/channels/:id/permissions/:role_id.
func (h *RoleHandler) DeleteChannelOverride(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel id")
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid role id")
	}

	if err := h.service.DeleteChannelOverride(c.Request().Context(), channelID, roleID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
