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
)

// BanHandler handles guild ban endpoints.
type BanHandler struct {
	guilds  database.GuildRepository
	members database.MemberRepository
	roles   database.RoleRepository
	bans    database.BanRepository
	gateway gateway.Dispatcher
}

// NewBanHandler creates a BanHandler.
func NewBanHandler(
	guilds database.GuildRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	bans database.BanRepository,
	gw gateway.Dispatcher,
) *BanHandler {
	return &BanHandler{
		guilds:  guilds,
		members: members,
		roles:   roles,
		bans:    bans,
		gateway: gw,
	}
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
	ctx := c.Request().Context()

	// Cannot ban self.
	if userID == targetUserID {
		return Error(c, http.StatusBadRequest, "CANNOT_BAN_SELF", "you cannot ban yourself")
	}

	if err := h.requirePermission(c, guildID, userID, permissions.PermBanMembers); err != nil || c.Response().Committed {
		return err
	}

	// Cannot ban guild owner.
	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}
	if guild.OwnerID == targetUserID {
		return Error(c, http.StatusForbidden, "FORBIDDEN", "cannot ban the guild owner")
	}

	// Check role hierarchy: banning user's highest role must be above target's highest role.
	callerRoles, err := h.roles.GetByMember(ctx, guildID, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	targetRoles, err := h.roles.GetByMember(ctx, guildID, targetUserID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if guild.OwnerID != userID && highestPosition(targetRoles) >= highestPosition(callerRoles) {
		return Error(c, http.StatusForbidden, "ROLE_HIERARCHY", "your highest role must be above the target's highest role")
	}

	var req banMemberRequest
	_ = c.Bind(&req) // optional body, ignore bind errors

	ban := &models.Ban{
		GuildID:   guildID,
		UserID:    targetUserID,
		Reason:    req.Reason,
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}

	if err := h.bans.Create(ctx, ban); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Auto-kick the member.
	_ = h.members.Delete(ctx, guildID, targetUserID)

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildBanAdd, ban)
	h.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberRemove, map[string]any{"guild_id": guildID, "user_id": targetUserID})

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
	ctx := c.Request().Context()

	if err := h.requirePermission(c, guildID, userID, permissions.PermBanMembers); err != nil || c.Response().Committed {
		return err
	}

	if err := h.bans.Delete(ctx, guildID, targetUserID); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildBanRemove, map[string]any{"guild_id": guildID, "user_id": targetUserID})

	return c.NoContent(http.StatusNoContent)
}

// ListBans handles GET /api/v1/guilds/:id/bans.
func (h *BanHandler) ListBans(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	if err := h.requirePermission(c, guildID, userID, permissions.PermBanMembers); err != nil || c.Response().Committed {
		return err
	}

	bans, err := h.bans.GetByGuildID(c.Request().Context(), guildID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if bans == nil {
		bans = []models.Ban{}
	}
	return c.JSON(http.StatusOK, bans)
}

// requirePermission checks that the user has the given permission in the guild.
func (h *BanHandler) requirePermission(c echo.Context, guildID, userID int64, perm permissions.Permission) error {
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

	computed := permissions.ComputeBasePermissions(everyoneRole, memberRoles)
	if !computed.Has(perm) {
		return Error(c, http.StatusForbidden, "MISSING_PERMISSIONS", "you do not have the required permissions")
	}

	return nil
}

// highestPosition returns the highest role position from a list of roles.
func highestPosition(roles []models.Role) int {
	max := -1
	for _, r := range roles {
		if r.Position > max {
			max = r.Position
		}
	}
	return max
}
