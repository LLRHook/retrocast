package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

func TestStorageUploadCompensationAndPermissions(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermAttachFiles)
	perms := service.NewPermissionChecker(guilds, members, roles, overrides)
	deleted := 0
	store := &mockStorage{DeleteFn: func(context.Context, string) error { deleted++; return nil }}
	repo := &mockAttachmentRepo{CreateFn: func(context.Context, *models.Attachment) error { return errors.New("metadata unavailable") }}
	svc := service.NewUploadService(repo, channelMock(), testSnowflake(), store, perms)
	if _, err := svc.UploadFile(context.Background(), testChannelID, testUserID, "proof.txt", 1, "text/plain", strings.NewReader("x")); err == nil || deleted != 1 {
		t.Fatalf("compensation: deleted=%d err=%v", deleted, err)
	}
	for _, size := range []int64{10 << 20, (10 << 20) + 1} {
		repo.CreateFn = func(_ context.Context, a *models.Attachment) error {
			if a.MessageID != nil || a.UploaderID != testUserID || a.ChannelID != testChannelID {
				t.Fatal("incorrect pending ownership")
			}
			return nil
		}
		_, err := svc.UploadFile(context.Background(), testChannelID, testUserID, "proof.txt", size, "text/plain", strings.NewReader("x"))
		if (size <= 10<<20) != (err == nil) {
			t.Fatalf("size boundary %d: %v", size, err)
		}
	}
	guilds, members, roles, overrides = permMocks(permissions.PermViewChannel | permissions.PermSendMessages)
	perms = service.NewPermissionChecker(guilds, members, roles, overrides)
	wrote := false
	messages := &mockMessageRepo{CreateFn: func(context.Context, *models.Message) error { wrote = true; return nil }}
	messageSvc := service.NewMessageService(messages, channelMock(), &mockDMChannelRepo{}, testSnowflake(), &mockGateway{}, perms)
	if _, err := messageSvc.SendMessage(context.Background(), testChannelID, testUserID, "with file", 123); !errors.Is(err, service.ErrForbidden) || wrote {
		t.Fatalf("ATTACH_FILES must be rechecked: wrote=%v err=%v", wrote, err)
	}
}
