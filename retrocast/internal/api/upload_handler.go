package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

const maxUploadSize = 10 << 20 // 10 MB

var allowedContentTypes = map[string]bool{
	"image/jpeg":       true,
	"image/png":        true,
	"image/gif":        true,
	"image/webp":       true,
	"application/pdf":  true,
	"text/plain":       true,
}

// FileStorage abstracts object storage operations for testability.
type FileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	GetURL(key string) string
	Delete(ctx context.Context, key string) error
}

// UploadHandler handles file upload endpoints.
type UploadHandler struct {
	attachments database.AttachmentRepository
	channels    database.ChannelRepository
	members     database.MemberRepository
	roles       database.RoleRepository
	guilds      database.GuildRepository
	overrides   database.ChannelOverrideRepository
	snowflake   *snowflake.Generator
	storage     FileStorage
}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler(
	attachments database.AttachmentRepository,
	channels database.ChannelRepository,
	members database.MemberRepository,
	roles database.RoleRepository,
	guilds database.GuildRepository,
	overrides database.ChannelOverrideRepository,
	sf *snowflake.Generator,
	storage FileStorage,
) *UploadHandler {
	return &UploadHandler{
		attachments: attachments,
		channels:    channels,
		members:     members,
		roles:       roles,
		guilds:      guilds,
		overrides:   overrides,
		snowflake:   sf,
		storage:     storage,
	}
}

// Upload handles POST /api/v1/channels/:id/attachments.
func (h *UploadHandler) Upload(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if channel == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "channel not found")
	}

	if err := h.requirePermission(c, channel.GuildID, channelID, userID, permissions.PermAttachFiles); err != nil {
		return err
	}

	file, err := c.FormFile("file")
	if err != nil {
		return Error(c, http.StatusBadRequest, "MISSING_FILE", "file field is required")
	}

	if file.Size > maxUploadSize {
		return Error(c, http.StatusBadRequest, "FILE_TOO_LARGE", "file must be under 10 MB")
	}

	contentType := file.Header.Get("Content-Type")
	if !isAllowedContentType(contentType) {
		return Error(c, http.StatusBadRequest, "INVALID_CONTENT_TYPE", "file type not allowed")
	}

	src, err := file.Open()
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	defer src.Close()

	attachmentID := h.snowflake.Generate().Int64()
	filename := filepath.Base(file.Filename)
	storageKey := fmt.Sprintf("attachments/%d/%d/%s", channelID, attachmentID, filename)

	if err := h.storage.Upload(ctx, storageKey, src, file.Size, contentType); err != nil {
		return Error(c, http.StatusInternalServerError, "UPLOAD_FAILED", "failed to upload file")
	}

	attachment := &models.Attachment{
		ID:          attachmentID,
		MessageID:   0, // not yet associated with a message
		Filename:    filename,
		ContentType: contentType,
		Size:        file.Size,
		StorageKey:  storageKey,
		URL:         h.storage.GetURL(storageKey),
	}

	if err := h.attachments.Create(ctx, attachment); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusCreated, attachment)
}

func isAllowedContentType(ct string) bool {
	if allowedContentTypes[ct] {
		return true
	}
	// Allow any image/* subtype.
	if strings.HasPrefix(ct, "image/") {
		return true
	}
	return false
}

// requirePermission checks that the user has the given permission in the channel.
func (h *UploadHandler) requirePermission(c echo.Context, guildID, channelID, userID int64, perm permissions.Permission) error {
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

	basePerms := permissions.ComputeBasePermissions(everyoneRole, memberRoles)

	channelOverrides, err := h.overrides.GetByChannel(ctx, channelID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	var everyoneOverride *models.ChannelOverride
	var roleOverrides []models.ChannelOverride

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

	computed := permissions.ComputeChannelPermissions(basePerms, everyoneOverride, roleOverrides)
	if !computed.Has(perm) {
		return Error(c, http.StatusForbidden, "MISSING_PERMISSIONS", "you do not have the required permissions")
	}

	return nil
}
