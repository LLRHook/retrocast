package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

const maxUploadSize = 10 << 20 // 10 MB

var allowedContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
}

// FileStorage abstracts object storage operations for testability.
type FileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	GetURL(key string) string
	Delete(ctx context.Context, key string) error
}

// UploadService handles file upload business logic.
type UploadService struct {
	attachments database.AttachmentRepository
	channels    database.ChannelRepository
	snowflake   *snowflake.Generator
	storage     FileStorage
	perms       *PermissionChecker
}

// NewUploadService creates an UploadService.
func NewUploadService(
	attachments database.AttachmentRepository,
	channels database.ChannelRepository,
	sf *snowflake.Generator,
	storage FileStorage,
	perms *PermissionChecker,
) *UploadService {
	return &UploadService{
		attachments: attachments,
		channels:    channels,
		snowflake:   sf,
		storage:     storage,
		perms:       perms,
	}
}

// UploadFile uploads a file to a channel.
func (s *UploadService) UploadFile(ctx context.Context, channelID, userID int64, filename string, size int64, contentType string, reader io.Reader) (*models.Attachment, error) {
	channel, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if channel == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermAttachFiles); err != nil {
		return nil, err
	}

	if size > maxUploadSize {
		return nil, BadRequest("FILE_TOO_LARGE", "file must be under 10 MB")
	}

	if !isAllowedContentType(contentType) {
		return nil, BadRequest("INVALID_CONTENT_TYPE", "file type not allowed")
	}

	attachmentID := s.snowflake.Generate().Int64()
	cleanFilename := filepath.Base(filename)
	storageKey := fmt.Sprintf("attachments/%d/%d/%s", channelID, attachmentID, cleanFilename)

	if err := s.storage.Upload(ctx, storageKey, reader, size, contentType); err != nil {
		return nil, NewError(ErrInternal, "UPLOAD_FAILED", "failed to upload file")
	}

	attachment := &models.Attachment{
		ID:          attachmentID,
		MessageID:   0,
		Filename:    cleanFilename,
		ContentType: contentType,
		Size:        size,
		StorageKey:  storageKey,
		URL:         s.storage.GetURL(storageKey),
	}

	if err := s.attachments.Create(ctx, attachment); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return attachment, nil
}

func isAllowedContentType(ct string) bool {
	if allowedContentTypes[ct] {
		return true
	}
	if strings.HasPrefix(ct, "image/") {
		return true
	}
	return false
}
