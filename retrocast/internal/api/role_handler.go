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

// RoleHandler handles role and channel override endpoints.
type RoleHandler struct {
	guilds    database.GuildRepository
	roles     database.RoleRepository
	members   database.MemberRepository
	channels  database.ChannelRepository
	overrides database.ChannelOverrideRepository
	snowflake *snowflake.Generator
	gateway   gateway.Dispatcher
}

// NewRoleHandler creates a RoleHandler.
func NewRoleHandler(
	guilds database.GuildRepository,
	roles database.RoleRepository,
	members database.MemberRepository,
	channels database.ChannelRepository,
	overrides database.ChannelOverrideRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
) *RoleHandler {
	return &RoleHandler{
		guilds:    guilds,
		roles:     roles,
		members:   members,
		channels:  channels,
		overrides: overrides,
		snowflake: sf,
		gateway:   gw,
	}
}

// highestRolePosition returns the highest (numerically largest) position among
// the acting user's roles. Returns 0 if the user has no assigned roles (only @everyone).
func (h *RoleHandler) highestRolePosition(c echo.Context, guildID, userID int64) (int, error) {
	memberRoles, err := h.roles.GetByMember(c.Request().Context(), guildID, userID)
	if err != nil {
		return 0, err
	}
	highest := 0
	for _, r := range memberRoles {
		if r.Position > highest {
			highest = r.Position
		}
	}
	return highest, nil
}

// isGuildOwner returns true if userID is the owner of the given guild.
func (h *RoleHandler) isGuildOwner(c echo.Context, guildID, userID int64) (bool, error) {
	guild, err := h.guilds.GetByID(c.Request().Context(), guildID)
	if err != nil {
		return false, err
	}
	if guild == nil {
		return false, nil
	}
	return guild.OwnerID == userID, nil
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

	if req.Name == "" || len(req.Name) > 100 {
		return errorJSON(c, http.StatusBadRequest, "INVALID_NAME", "name must be 1-100 characters")
	}

	actorID := auth.GetUserID(c)

	// Role hierarchy: only guild owner can create roles at or above their own position.
	isOwner, err := h.isGuildOwner(c, guildID, actorID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if !isOwner {
		highest, err := h.highestRolePosition(c, guildID, actorID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if req.Position >= highest {
			return errorJSON(c, http.StatusForbidden, "ROLE_HIERARCHY", "cannot create a role at or above your highest role position")
		}
	}

	role := &models.Role{
		ID:          h.snowflake.Generate().Int64(),
		GuildID:     guildID,
		Name:        req.Name,
		Color:       req.Color,
		Permissions: req.Permissions,
		Position:    req.Position,
	}

	if err := h.roles.Create(c.Request().Context(), role); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusCreated, role)
}

// ListRoles handles GET /api/v1/guilds/:id/roles.
func (h *RoleHandler) ListRoles(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
	}

	roles, err := h.roles.GetByGuildID(c.Request().Context(), guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if roles == nil {
		roles = []models.Role{}
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

	ctx := c.Request().Context()
	role, err := h.roles.GetByID(ctx, roleID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if role == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "role not found")
	}

	// Role hierarchy check.
	actorID := auth.GetUserID(c)
	isOwner, err := h.isGuildOwner(c, guildID, actorID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if !isOwner {
		highest, err := h.highestRolePosition(c, guildID, actorID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if role.Position >= highest {
			return errorJSON(c, http.StatusForbidden, "ROLE_HIERARCHY", "cannot modify a role at or above your highest role position")
		}
	}

	if req.Name != nil {
		if *req.Name == "" || len(*req.Name) > 100 {
			return errorJSON(c, http.StatusBadRequest, "INVALID_NAME", "name must be 1-100 characters")
		}
		role.Name = *req.Name
	}
	if req.Color != nil {
		role.Color = *req.Color
	}
	if req.Permissions != nil {
		role.Permissions = *req.Permissions
	}
	if req.Position != nil {
		role.Position = *req.Position
	}

	if err := h.roles.Update(ctx, role); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
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

	ctx := c.Request().Context()
	role, err := h.roles.GetByID(ctx, roleID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if role == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "role not found")
	}
	if role.IsDefault {
		return errorJSON(c, http.StatusForbidden, "CANNOT_DELETE", "cannot delete the @everyone role")
	}

	// Role hierarchy check.
	actorID := auth.GetUserID(c)
	isOwner, err := h.isGuildOwner(c, guildID, actorID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if !isOwner {
		highest, err := h.highestRolePosition(c, guildID, actorID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if role.Position >= highest {
			return errorJSON(c, http.StatusForbidden, "ROLE_HIERARCHY", "cannot delete a role at or above your highest role position")
		}
	}

	if err := h.roles.Delete(ctx, roleID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
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

	ctx := c.Request().Context()

	member, err := h.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "member not found")
	}

	role, err := h.roles.GetByID(ctx, roleID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if role == nil || role.GuildID != guildID {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "role not found")
	}

	// Role hierarchy check.
	actorID := auth.GetUserID(c)
	isOwner, err := h.isGuildOwner(c, guildID, actorID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if !isOwner {
		highest, err := h.highestRolePosition(c, guildID, actorID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if role.Position >= highest {
			return errorJSON(c, http.StatusForbidden, "ROLE_HIERARCHY", "cannot assign a role at or above your highest role position")
		}
	}

	if err := h.members.AddRole(ctx, guildID, userID, roleID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
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

	ctx := c.Request().Context()

	// Role hierarchy check.
	actorID := auth.GetUserID(c)
	isOwner, err := h.isGuildOwner(c, guildID, actorID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if !isOwner {
		role, err := h.roles.GetByID(ctx, roleID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		if role != nil {
			highest, err := h.highestRolePosition(c, guildID, actorID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
			if role.Position >= highest {
				return errorJSON(c, http.StatusForbidden, "ROLE_HIERARCHY", "cannot remove a role at or above your highest role position")
			}
		}
	}

	if err := h.members.RemoveRole(ctx, guildID, userID, roleID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
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

	ctx := c.Request().Context()

	ch, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if ch == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	override := &models.ChannelOverride{
		ChannelID: channelID,
		RoleID:    roleID,
		Allow:     req.Allow,
		Deny:      req.Deny,
	}

	if err := h.overrides.Set(ctx, override); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
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

	if err := h.overrides.Delete(c.Request().Context(), channelID, roleID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.NoContent(http.StatusNoContent)
}

// RegisterRoutes registers all role-related routes on the given groups.
// guildGroup should be "/api/v1/guilds" (authenticated).
// channelGroup should be "/api/v1/channels" (authenticated).
func (h *RoleHandler) RegisterRoutes(guildGroup, channelGroup *echo.Group) {
	// Role CRUD
	guildGroup.POST("/:id/roles", h.CreateRole)
	guildGroup.GET("/:id/roles", h.ListRoles)
	guildGroup.PATCH("/:id/roles/:role_id", h.UpdateRole)
	guildGroup.DELETE("/:id/roles/:role_id", h.DeleteRole)

	// Member role assignment
	guildGroup.PUT("/:id/members/:user_id/roles/:role_id", h.AssignRole)
	guildGroup.DELETE("/:id/members/:user_id/roles/:role_id", h.RemoveRole)

	// Channel overrides
	channelGroup.PUT("/:id/permissions/:role_id", h.SetChannelOverride)
	channelGroup.DELETE("/:id/permissions/:role_id", h.DeleteChannelOverride)
}
