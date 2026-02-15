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

// GuildHandler handles guild CRUD endpoints.
type GuildHandler struct {
	guilds    database.GuildRepository
	channels  database.ChannelRepository
	members   database.MemberRepository
	roles     database.RoleRepository
	snowflake *snowflake.Generator
	gateway   gateway.Dispatcher
}

// NewGuildHandler creates a GuildHandler.
func NewGuildHandler(
	guilds database.GuildRepository,
	channels database.ChannelRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
) *GuildHandler {
	return &GuildHandler{
		guilds:    guilds,
		channels:  channels,
		members:   members,
		roles:     roles,
		snowflake: sf,
		gateway:   gw,
	}
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
	if len(req.Name) < 2 || len(req.Name) > 100 {
		return errorJSON(c, http.StatusBadRequest, "INVALID_NAME", "guild name must be 2-100 characters")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)
	now := time.Now()

	guild := &models.Guild{
		ID:        h.snowflake.Generate().Int64(),
		Name:      req.Name,
		OwnerID:   userID,
		CreatedAt: now,
	}
	if err := h.guilds.Create(ctx, guild); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Create @everyone role with basic permissions.
	everyoneRole := &models.Role{
		ID:          h.snowflake.Generate().Int64(),
		GuildID:     guild.ID,
		Name:        "@everyone",
		Permissions: int64(permissions.PermViewChannel | permissions.PermSendMessages | permissions.PermReadMessageHistory),
		Position:    0,
		IsDefault:   true,
	}
	if err := h.roles.Create(ctx, everyoneRole); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Create admin role for the owner.
	adminRole := &models.Role{
		ID:          h.snowflake.Generate().Int64(),
		GuildID:     guild.ID,
		Name:        "Admin",
		Permissions: int64(permissions.PermAdministrator),
		Position:    1,
	}
	if err := h.roles.Create(ctx, adminRole); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Add the creator as a member.
	member := &models.Member{
		GuildID:  guild.ID,
		UserID:   userID,
		JoinedAt: now,
	}
	if err := h.members.Create(ctx, member); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Assign both roles to the owner.
	if err := h.members.AddRole(ctx, guild.ID, userID, everyoneRole.ID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if err := h.members.AddRole(ctx, guild.ID, userID, adminRole.ID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Auto-create #general text channel and General voice channel.
	generalText := &models.Channel{
		ID:       h.snowflake.Generate().Int64(),
		GuildID:  guild.ID,
		Name:     "general",
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := h.channels.Create(ctx, generalText); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	generalVoice := &models.Channel{
		ID:       h.snowflake.Generate().Int64(),
		GuildID:  guild.ID,
		Name:     "General",
		Type:     models.ChannelTypeVoice,
		Position: 1,
	}
	if err := h.channels.Create(ctx, generalVoice); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToUser(userID, gateway.EventGuildCreate, guild)
	return c.JSON(http.StatusCreated, map[string]any{"data": guild})
}

// GetGuild handles GET /api/v1/guilds/:id.
func (h *GuildHandler) GetGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	// Verify the user is a member.
	member, err := h.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
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

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	if err := h.requirePermission(c, guildID, userID, int64(permissions.PermManageGuild)); err != nil {
		return err
	}

	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	var req updateGuildRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if req.Name != nil {
		if len(*req.Name) < 2 || len(*req.Name) > 100 {
			return errorJSON(c, http.StatusBadRequest, "INVALID_NAME", "guild name must be 2-100 characters")
		}
		guild.Name = *req.Name
	}
	if req.Icon != nil {
		guild.IconHash = req.Icon
	}

	if err := h.guilds.Update(ctx, guild); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildUpdate, guild)
	return c.JSON(http.StatusOK, map[string]any{"data": guild})
}

// DeleteGuild handles DELETE /api/v1/guilds/:id.
func (h *GuildHandler) DeleteGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	// Only the guild owner can delete it.
	if guild.OwnerID != userID {
		return errorJSON(c, http.StatusForbidden, "FORBIDDEN", "only the guild owner can delete the guild")
	}

	if err := h.guilds.Delete(ctx, guildID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildDelete, map[string]any{"id": guildID})
	return c.NoContent(http.StatusNoContent)
}

// ListMyGuilds handles GET /api/v1/users/@me/guilds.
func (h *GuildHandler) ListMyGuilds(c echo.Context) error {
	userID := auth.GetUserID(c)

	guilds, err := h.guilds.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guilds == nil {
		guilds = []models.Guild{}
	}

	return c.JSON(http.StatusOK, map[string]any{"data": guilds})
}

// RequirePermission returns the permission-checking function for use by other handlers.
func (h *GuildHandler) RequirePermission() func(ctx echo.Context, guildID, userID, perm int64) error {
	return h.requirePermission
}

// requirePermission checks that the user has the given permission in a guild.
// Returns an echo error response if denied, or nil if allowed.
func (h *GuildHandler) requirePermission(ctx echo.Context, guildID, userID, perm int64) error {
	reqCtx := ctx.Request().Context()

	// Guild owner has all permissions.
	guild, err := h.guilds.GetByID(reqCtx, guildID)
	if err != nil {
		return errorJSON(ctx, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild == nil {
		return errorJSON(ctx, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == userID {
		return nil
	}

	// Check membership.
	member, err := h.members.GetByGuildAndUser(reqCtx, guildID, userID)
	if err != nil {
		return errorJSON(ctx, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(ctx, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	// Compute effective permissions from all member roles.
	roles, err := h.roles.GetByMember(reqCtx, guildID, userID)
	if err != nil {
		return errorJSON(ctx, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var perms int64
	for _, r := range roles {
		perms |= r.Permissions
	}

	// Administrator implies all permissions.
	if perms&int64(permissions.PermAdministrator) != 0 {
		return nil
	}

	if perms&perm == 0 {
		return errorJSON(ctx, http.StatusForbidden, "MISSING_PERMISSIONS", "you do not have permission to perform this action")
	}
	return nil
}
