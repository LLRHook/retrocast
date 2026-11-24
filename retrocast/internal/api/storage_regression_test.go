package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

func TestDMSendRejectsAttachmentIDs(t *testing.T) {
	gw := &mockGateway{}
	dms := &mockDMChannelRepo{
		GetByIDFn: func(context.Context, int64) (*models.DMChannel, error) {
			return &models.DMChannel{ID: testChannelID, Type: models.DMTypeDM}, nil
		},
		IsRecipientFn: func(context.Context, int64, int64) (bool, error) { return true, nil },
	}
	msgs := &mockMessageRepo{CreateFn: func(context.Context, *models.Message) error {
		t.Fatal("DM attachment rejection must precede message creation")
		return nil
	}}
	svc := service.NewMessageService(msgs, &mockChannelRepo{}, dms, testSnowflake(), gw, nil)
	handler := NewMessageHandler(svc)
	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(`{"content":"DM with file","attachment_ids":["123"]}`))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)
	if err := handler.SendMessage(c); err != nil {
		t.Fatal(err)
	}
	var body ErrorResponse
	if err := decodeResponse(rec, &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest || body.Error.Code != "INVALID_ATTACHMENTS" {
		t.Fatalf("DM attachment refusal: %d %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 0 {
		t.Fatal("refused DM attachment must not dispatch")
	}
}

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
