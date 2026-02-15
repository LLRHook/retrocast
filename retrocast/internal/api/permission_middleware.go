package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// RequireGuildPermission returns middleware that checks guild-level permissions.
// It expects the route to have a ":id" param for the guild ID.
func RequireGuildPermission(
	perm permissions.Permission,
	guilds database.GuildRepository,
	roles database.RoleRepository,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid guild id")
			}

			userID := auth.GetUserID(c)
			ctx := c.Request().Context()

			// Guild owner has all permissions.
			guild, err := guilds.GetByID(ctx, guildID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
			if guild == nil {
				return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
			}
			if guild.OwnerID == userID {
				return next(c)
			}

			// Fetch @everyone role and member's roles.
			allRoles, err := roles.GetByGuildID(ctx, guildID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}

			var everyoneRole *models.Role
			for i := range allRoles {
				if allRoles[i].IsDefault {
					everyoneRole = &allRoles[i]
					break
				}
			}
			if everyoneRole == nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "missing @everyone role")
			}

			memberRoles, err := roles.GetByMember(ctx, guildID, userID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}

			basePerms := permissions.ComputeBasePermissions(*everyoneRole, memberRoles)

			if !basePerms.Has(perm) {
				return errorJSON(c, http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action")
			}

			return next(c)
		}
	}
}

// RequireChannelPermission returns middleware that checks channel-level permissions
// (including channel overrides). It expects the route to have a ":id" param for the channel ID.
func RequireChannelPermission(
	perm permissions.Permission,
	guilds database.GuildRepository,
	channels database.ChannelRepository,
	roles database.RoleRepository,
	overrides database.ChannelOverrideRepository,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel id")
			}

			userID := auth.GetUserID(c)
			ctx := c.Request().Context()

			// Look up channel to get its guild.
			channel, err := channels.GetByID(ctx, channelID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
			if channel == nil {
				return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
			}

			guildID := channel.GuildID

			// Guild owner has all permissions.
			guild, err := guilds.GetByID(ctx, guildID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
			if guild == nil {
				return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "guild not found")
			}
			if guild.OwnerID == userID {
				return next(c)
			}

			// Fetch all guild roles to find @everyone.
			allRoles, err := roles.GetByGuildID(ctx, guildID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}

			var everyoneRole *models.Role
			for i := range allRoles {
				if allRoles[i].IsDefault {
					everyoneRole = &allRoles[i]
					break
				}
			}
			if everyoneRole == nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "missing @everyone role")
			}

			// Get member's assigned roles.
			memberRoles, err := roles.GetByMember(ctx, guildID, userID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}

			basePerms := permissions.ComputeBasePermissions(*everyoneRole, memberRoles)

			// Fetch channel overrides.
			channelOverrides, err := overrides.GetByChannel(ctx, channelID)
			if err != nil {
				return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}

			// Separate @everyone override from role-specific overrides.
			var everyoneOverride *models.ChannelOverride
			var roleOverrides []models.ChannelOverride

			// Build a set of member's role IDs for fast lookup.
			memberRoleIDs := make(map[int64]bool, len(memberRoles))
			for _, r := range memberRoles {
				memberRoleIDs[r.ID] = true
			}

			for i := range channelOverrides {
				if channelOverrides[i].RoleID == everyoneRole.ID {
					everyoneOverride = &channelOverrides[i]
				} else if memberRoleIDs[channelOverrides[i].RoleID] {
					roleOverrides = append(roleOverrides, channelOverrides[i])
				}
			}

			channelPerms := permissions.ComputeChannelPermissions(basePerms, everyoneOverride, roleOverrides)

			if !channelPerms.Has(perm) {
				return errorJSON(c, http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action")
			}

			return next(c)
		}
	}
}
