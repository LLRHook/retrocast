package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

// newReadStateHandler wires up a ReadStateHandler with mocks via the service layer.
func newReadStateHandler(
	rs *mockReadStateRepo,
	chs *mockChannelRepo,
	dms *mockDMChannelRepo,
	guilds *mockGuildRepo,
	members *mockMemberRepo,
	roles *mockRoleRepo,
	overrides *mockChannelOverrideRepo,
) *ReadStateHandler {
	perms := service.NewPermissionChecker(guilds, members, roles, overrides)
	svc := service.NewReadStateService(rs, chs, dms, perms)
	return NewReadStateHandler(svc)
}

// ---------------------------------------------------------------------------
// Ack tests
// ---------------------------------------------------------------------------

func TestAckReadState_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()

	var capturedUserID, capturedChannelID, capturedMsgID int64
	rs := &mockReadStateRepo{
		UpsertFn: func(_ context.Context, userID, channelID, lastMessageID int64) error {
			capturedUserID = userID
			capturedChannelID = channelID
			capturedMsgID = lastMessageID
			return nil
		},
	}

	h := newReadStateHandler(rs, channels, &mockDMChannelRepo{}, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/ack/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	err := h.Ack(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedUserID != testUserID {
		t.Fatalf("expected userID %d, got %d", testUserID, capturedUserID)
	}
	if capturedChannelID != testChannelID {
		t.Fatalf("expected channelID %d, got %d", testChannelID, capturedChannelID)
	}
	if capturedMsgID != testMsgID {
		t.Fatalf("expected msgID %d, got %d", testMsgID, capturedMsgID)
	}
}

func TestAckReadState_InvalidChannelID(t *testing.T) {
	h := newReadStateHandler(
		&mockReadStateRepo{}, &mockChannelRepo{}, &mockDMChannelRepo{},
		&mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockChannelOverrideRepo{},
	)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/abc/ack/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("abc", "5000")
	setAuthUser(c, testUserID)

	_ = h.Ack(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAckReadState_InvalidMessageID(t *testing.T) {
	h := newReadStateHandler(
		&mockReadStateRepo{}, &mockChannelRepo{}, &mockDMChannelRepo{},
		&mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockChannelOverrideRepo{},
	)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/ack/abc", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "abc")
	setAuthUser(c, testUserID)

	_ = h.Ack(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAckReadState_ChannelNotFound(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.Channel, error) {
			return nil, nil
		},
	}
	dms := &mockDMChannelRepo{}

	h := newReadStateHandler(&mockReadStateRepo{}, channels, dms, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/9999/ack/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("9999", "5000")
	setAuthUser(c, testUserID)

	_ = h.Ack(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAckReadState_NoPermission(t *testing.T) {
	// User has no ViewChannel permission.
	guilds, members, roles, overrides := permMocks(0)
	channels := channelMock()

	h := newReadStateHandler(&mockReadStateRepo{}, channels, &mockDMChannelRepo{}, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/ack/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	_ = h.Ack(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetReadStates tests
// ---------------------------------------------------------------------------

func TestGetReadStates_Success(t *testing.T) {
	now := time.Now()
	rs := &mockReadStateRepo{
		GetByUserFn: func(_ context.Context, userID int64) ([]models.ReadState, error) {
			return []models.ReadState{
				{UserID: userID, ChannelID: testChannelID, LastMessageID: testMsgID, MentionCount: 0, UpdatedAt: now},
			}, nil
		},
	}

	h := newReadStateHandler(rs, &mockChannelRepo{}, &mockDMChannelRepo{},
		&mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockChannelOverrideRepo{})

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me/read-states", nil)
	setAuthUser(c, testUserID)

	err := h.GetReadStates(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result []models.ReadState
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 read state, got %d", len(result))
	}
	if result[0].ChannelID != testChannelID {
		t.Fatalf("expected channel_id %d, got %d", testChannelID, result[0].ChannelID)
	}
}

func TestGetReadStates_Empty(t *testing.T) {
	rs := &mockReadStateRepo{
		GetByUserFn: func(_ context.Context, _ int64) ([]models.ReadState, error) {
			return nil, nil
		},
	}

	h := newReadStateHandler(rs, &mockChannelRepo{}, &mockDMChannelRepo{},
		&mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockChannelOverrideRepo{})

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me/read-states", nil)
	setAuthUser(c, testUserID)

	err := h.GetReadStates(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result []models.ReadState
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 read states, got %d", len(result))
	}
}
