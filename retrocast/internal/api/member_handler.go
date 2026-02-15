package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// MemberHandler handles member endpoints.
type MemberHandler struct {
	members   database.MemberRepository
	guilds    database.GuildRepository
	roles     database.RoleRepository
	guildPerm func(ctx echo.Context, guildID, userID, perm int64) error
	gateway   gateway.Dispatcher
}

// NewMemberHandler creates a MemberHandler.
func NewMemberHandler(
	members database.MemberRepository,
	guilds database.GuildRepository,
	roles database.RoleRepository,
	guildPerm func(ctx echo.Context, guildID, userID, perm int64) error,
	gw gateway.Dispatcher,
) *MemberHandler {
	return &MemberHandler{
		members:   members,
		guilds:    guilds,
		roles:     roles,
		guildPerm: guildPerm,
		gateway:   gw,
	}
}

// ListMembers handles GET /api/v1/guilds/:id/members.
func (h *MemberHandler) ListMembers(c echo.Context) error {
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

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	members, err := h.members.GetByGuildID(ctx, guildID, limit, offset)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if members == nil {
		members = []models.Member{}
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

	ctx := c.Request().Context()
	callerID := auth.GetUserID(c)

	// Verify caller is a member.
	callerMember, err := h.members.GetByGuildAndUser(ctx, guildID, callerID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if callerMember == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	member, err := h.members.GetByGuildAndUser(ctx, guildID, targetUserID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "member not found")
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

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	member, err := h.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
	}

	var req updateMemberRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if req.Nickname != nil {
		if len(*req.Nickname) > 32 {
			return errorJSON(c, http.StatusBadRequest, "INVALID_NICKNAME", "nickname must be 32 characters or fewer")
		}
		if *req.Nickname == "" {
			member.Nickname = nil
		} else {
			member.Nickname = req.Nickname
		}
	}

	if err := h.members.Update(ctx, member); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberUpdate, member)
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

	ctx := c.Request().Context()
	callerID := auth.GetUserID(c)

	var req updateMemberRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Check required permissions based on what's being changed.
	if req.Nickname != nil {
		if err := h.guildPerm(c, guildID, callerID, int64(permissions.PermManageNicknames)); err != nil {
			return err
		}
	}
	if req.Roles != nil {
		if err := h.guildPerm(c, guildID, callerID, int64(permissions.PermManageRoles)); err != nil {
			return err
		}
	}

	member, err := h.members.GetByGuildAndUser(ctx, guildID, targetUserID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "member not found")
	}

	if req.Nickname != nil {
		if len(*req.Nickname) > 32 {
			return errorJSON(c, http.StatusBadRequest, "INVALID_NICKNAME", "nickname must be 32 characters or fewer")
		}
		if *req.Nickname == "" {
			member.Nickname = nil
		} else {
			member.Nickname = req.Nickname
		}
		if err := h.members.Update(ctx, member); err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
	}

	if req.Roles != nil {
		// Remove all current non-default roles and apply new ones.
		guildRoles, err := h.roles.GetByGuildID(ctx, guildID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}

		// Build lookup of valid guild role IDs.
		validRoles := make(map[int64]bool, len(guildRoles))
		for _, r := range guildRoles {
			if !r.IsDefault {
				validRoles[r.ID] = true
			}
		}

		// Remove existing non-default roles.
		for _, roleID := range member.Roles {
			if validRoles[roleID] {
				if err := h.members.RemoveRole(ctx, guildID, targetUserID, roleID); err != nil {
					return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
				}
			}
		}

		// Add requested roles.
		for _, roleID := range *req.Roles {
			if !validRoles[roleID] {
				return errorJSON(c, http.StatusBadRequest, "INVALID_ROLE", "invalid role ID")
			}
			if err := h.members.AddRole(ctx, guildID, targetUserID, roleID); err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
		}

		// Re-fetch the member to get updated roles.
		member, err = h.members.GetByGuildAndUser(ctx, guildID, targetUserID)
		if err != nil {
			return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberUpdate, member)
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

	ctx := c.Request().Context()
	callerID := auth.GetUserID(c)

	if err := h.guildPerm(c, guildID, callerID, int64(permissions.PermKickMembers)); err != nil {
		return err
	}

	// Cannot kick the guild owner.
	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild != nil && guild.OwnerID == targetUserID {
		return errorJSON(c, http.StatusForbidden, "FORBIDDEN", "cannot kick the guild owner")
	}

	member, err := h.members.GetByGuildAndUser(ctx, guildID, targetUserID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "member not found")
	}

	if err := h.members.Delete(ctx, guildID, targetUserID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberRemove, map[string]any{"guild_id": guildID, "user_id": targetUserID})
	return c.NoContent(http.StatusNoContent)
}

// LeaveGuild handles DELETE /api/v1/guilds/:id/members/@me.
func (h *MemberHandler) LeaveGuild(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	ctx := c.Request().Context()
	userID := auth.GetUserID(c)

	// Cannot leave a guild you own.
	guild, err := h.guilds.GetByID(ctx, guildID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if guild != nil && guild.OwnerID == userID {
		return errorJSON(c, http.StatusForbidden, "FORBIDDEN", "guild owner cannot leave; transfer ownership or delete the guild")
	}

	member, err := h.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if member == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "you are not a member of this guild")
	}

	if err := h.members.Delete(ctx, guildID, userID); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(guildID, gateway.EventGuildMemberRemove, map[string]any{"guild_id": guildID, "user_id": userID})
	return c.NoContent(http.StatusNoContent)
}
