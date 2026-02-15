package api

import (
	"crypto/rand"
	"encoding/hex"
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

// InviteHandler handles invite endpoints.
type InviteHandler struct {
	invites database.InviteRepository
	guilds  database.GuildRepository
	members database.MemberRepository
	roles   database.RoleRepository
	bans    database.BanRepository
	gateway gateway.Dispatcher
}

// NewInviteHandler creates an InviteHandler.
func NewInviteHandler(
	invites database.InviteRepository,
	guilds database.GuildRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	bans database.BanRepository,
	gw gateway.Dispatcher,
) *InviteHandler {
	return &InviteHandler{
		invites: invites,
		guilds:  guilds,
		members: members,
		roles:   roles,
		bans:    bans,
		gateway: gw,
	}
}

type createInviteRequest struct {
	MaxUses       int `json:"max_uses"`
	MaxAgeSeconds int `json:"max_age_seconds"`
}

type inviteInfoResponse struct {
	Code        string `json:"code"`
	GuildName   string `json:"guild_name"`
	MemberCount int    `json:"member_count"`
	CreatorID   int64  `json:"creator_id,string"`
}

// CreateInvite handles POST /api/v1/guilds/:id/invites.
func (h *InviteHandler) CreateInvite(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	if err := h.requirePermission(c, guildID, userID, permissions.PermCreateInvite); err != nil {
		return err
	}

	var req createInviteRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Default max_age to 24 hours if not specified.
	if req.MaxAgeSeconds == 0 {
		req.MaxAgeSeconds = 86400
	}

	code, err := generateInviteCode()
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	now := time.Now()
	var expiresAt *time.Time
	if req.MaxAgeSeconds > 0 {
		t := now.Add(time.Duration(req.MaxAgeSeconds) * time.Second)
		expiresAt = &t
	}

	invite := &models.Invite{
		Code:      code,
		GuildID:   guildID,
		CreatorID: userID,
		MaxUses:   req.MaxUses,
		Uses:      0,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	if err := h.invites.Create(ctx, invite); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
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

	if err := h.requirePermission(c, guildID, userID, permissions.PermManageGuild); err != nil {
		return err
	}

	invites, err := h.invites.GetByGuildID(c.Request().Context(), guildID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if invites == nil {
		invites = []models.Invite{}
	}
	return c.JSON(http.StatusOK, invites)
}

// GetInvite handles GET /api/v1/invites/:code (no auth required).
func (h *InviteHandler) GetInvite(c echo.Context) error {
	code := c.Param("code")
	ctx := c.Request().Context()

	invite, err := h.invites.GetByCode(ctx, code)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if invite == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "invite not found")
	}

	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return Error(c, http.StatusNotFound, "EXPIRED", "invite has expired")
	}

	guild, err := h.guilds.GetByID(ctx, invite.GuildID)
	if err != nil || guild == nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// Get approximate member count.
	members, err := h.members.GetByGuildID(ctx, invite.GuildID, 10000, 0)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusOK, inviteInfoResponse{
		Code:        invite.Code,
		GuildName:   guild.Name,
		MemberCount: len(members),
		CreatorID:   invite.CreatorID,
	})
}

// AcceptInvite handles POST /api/v1/invites/:code (auth required).
func (h *InviteHandler) AcceptInvite(c echo.Context) error {
	code := c.Param("code")
	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	invite, err := h.invites.GetByCode(ctx, code)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if invite == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "invite not found")
	}

	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return Error(c, http.StatusGone, "EXPIRED", "invite has expired")
	}

	if invite.MaxUses > 0 && invite.Uses >= invite.MaxUses {
		return Error(c, http.StatusGone, "MAX_USES", "invite has reached maximum uses")
	}

	// Check if already a member.
	existing, err := h.members.GetByGuildAndUser(ctx, invite.GuildID, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if existing != nil {
		return Error(c, http.StatusConflict, "ALREADY_MEMBER", "you are already a member of this guild")
	}

	// Check if user is banned.
	ban, err := h.bans.GetByGuildAndUser(ctx, invite.GuildID, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if ban != nil {
		return Error(c, http.StatusForbidden, "BANNED", "you are banned from this guild")
	}

	member := &models.Member{
		GuildID:  invite.GuildID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	if err := h.members.Create(ctx, member); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if err := h.invites.IncrementUses(ctx, code); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	guild, err := h.guilds.GetByID(ctx, invite.GuildID)
	if err != nil || guild == nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	h.gateway.DispatchToGuild(invite.GuildID, gateway.EventGuildMemberAdd, member)

	return c.JSON(http.StatusOK, guild)
}

// RevokeInvite handles DELETE /api/v1/invites/:code.
func (h *InviteHandler) RevokeInvite(c echo.Context) error {
	code := c.Param("code")
	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	invite, err := h.invites.GetByCode(ctx, code)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if invite == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "invite not found")
	}

	// Creator can always revoke, otherwise need MANAGE_GUILD.
	if invite.CreatorID != userID {
		if err := h.requirePermission(c, invite.GuildID, userID, permissions.PermManageGuild); err != nil {
			return err
		}
	}

	if err := h.invites.Delete(ctx, code); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.NoContent(http.StatusNoContent)
}

// requirePermission checks that the user has the given permission in the guild.
func (h *InviteHandler) requirePermission(c echo.Context, guildID, userID int64, perm permissions.Permission) error {
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

// generateInviteCode returns a random 8-character hex code.
func generateInviteCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
