package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// ---------------------------------------------------------------------------
// Shared test constants and helpers for message handler tests
// ---------------------------------------------------------------------------

const (
	testGuildID   int64 = 1000
	testChannelID int64 = 2000
	testUserID    int64 = 3000
	testOwnerID   int64 = 9999
	testMsgID     int64 = 5000
	testRoleID    int64 = 6000
)

// newMessageHandler wires up a MessageHandler with the given mocks.
func newMessageHandler(
	msgs *mockMessageRepo,
	chs *mockChannelRepo,
	mems *mockMemberRepo,
	roles *mockRoleRepo,
	guilds *mockGuildRepo,
	overrides *mockChannelOverrideRepo,
	gw *mockGateway,
) *MessageHandler {
	return NewMessageHandler(msgs, chs, &mockDMChannelRepo{}, mems, roles, guilds, overrides, testSnowflake(), gw)
}

// permMocks sets up the standard guild/member/role/override mocks so that a
// non-owner member passes permission checks with the given @everyone perms.
// No channel overrides by default.
func permMocks(everyonePerms permissions.Permission) (*mockGuildRepo, *mockMemberRepo, *mockRoleRepo, *mockChannelOverrideRepo) {
	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: testGuildID, OwnerID: testOwnerID}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(_ context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: time.Now()}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(_ context.Context, _, _ int64) ([]models.Role, error) {
			return nil, nil // no extra roles
		},
		GetByGuildIDFn: func(_ context.Context, _ int64) ([]models.Role, error) {
			return []models.Role{
				{ID: testRoleID, GuildID: testGuildID, Name: "@everyone", Permissions: int64(everyonePerms), IsDefault: true},
			}, nil
		},
	}
	overrides := &mockChannelOverrideRepo{
		GetByChannelFn: func(_ context.Context, _ int64) ([]models.ChannelOverride, error) {
			return nil, nil // no overrides
		},
	}
	return guilds, members, roles, overrides
}

func channelMock() *mockChannelRepo {
	return &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			return &models.Channel{ID: testChannelID, GuildID: testGuildID, Name: "general"}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// SendMessage tests
// ---------------------------------------------------------------------------

func TestSendMessage_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermSendMessages | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}

	created := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
			Content: "hello", CreatedAt: time.Now(),
		},
		AuthorUsername: "testuser",
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return created, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(`{"content":"hello"}`))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	err := h.SendMessage(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 1 || gw.events[0].Event != gateway.EventMessageCreate {
		t.Fatalf("expected MESSAGE_CREATE event, got %+v", gw.events)
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermSendMessages | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(`{"content":""}`))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.SendMessage(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSendMessage_TooLongContent(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermSendMessages | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	longContent := strings.Repeat("a", 2001)
	body := `{"content":"` + longContent + `"}`
	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(body))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.SendMessage(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSendMessage_NoPermission(t *testing.T) {
	// @everyone has ViewChannel but NOT SendMessages.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(`{"content":"hello"}`))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.SendMessage(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSendMessage_ChannelOverrideDeny(t *testing.T) {
	// @everyone role grants SendMessages at guild level, but channel override denies it.
	guilds, members, roles, overrides := permMocks(permissions.PermSendMessages | permissions.PermViewChannel)

	overrides.GetByChannelFn = func(_ context.Context, _ int64) ([]models.ChannelOverride, error) {
		return []models.ChannelOverride{
			{ChannelID: testChannelID, RoleID: testRoleID, Allow: 0, Deny: int64(permissions.PermSendMessages)},
		}, nil
	}

	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(`{"content":"hello"}`))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.SendMessage(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (channel override deny), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSendMessage_ChannelOverrideAllow(t *testing.T) {
	// @everyone role does NOT have SendMessages at guild level, but channel override allows it.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel) // no SendMessages in base

	overrides.GetByChannelFn = func(_ context.Context, _ int64) ([]models.ChannelOverride, error) {
		return []models.ChannelOverride{
			{ChannelID: testChannelID, RoleID: testRoleID, Allow: int64(permissions.PermSendMessages), Deny: 0},
		}, nil
	}

	channels := channelMock()
	gw := &mockGateway{}
	created := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
			Content: "allowed by override", CreatedAt: time.Now(),
		},
		AuthorUsername: "testuser",
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return created, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/messages", strings.NewReader(`{"content":"allowed by override"}`))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	err := h.SendMessage(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 (channel override allow), got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetMessages tests
// ---------------------------------------------------------------------------

func TestGetMessages_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}

	var capturedLimit int
	msgs := &mockMessageRepo{
		GetByChannelIDFn: func(_ context.Context, _ int64, _ *int64, limit int) ([]models.MessageWithAuthor, error) {
			capturedLimit = limit
			return []models.MessageWithAuthor{
				{Message: models.Message{ID: 1, ChannelID: testChannelID, AuthorID: testUserID, Content: "msg1", CreatedAt: time.Now()}},
			}, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages?limit=25", nil)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	c.QueryParams().Set("limit", "25")
	setAuthUser(c, testUserID)

	err := h.GetMessages(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedLimit != 25 {
		t.Fatalf("expected limit 25, got %d", capturedLimit)
	}

	var result []models.MessageWithAuthor
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestGetMessages_NoPermission(t *testing.T) {
	// Has ViewChannel but NOT ReadMessageHistory.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages", nil)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.GetMessages(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetMessage tests
// ---------------------------------------------------------------------------

func TestGetMessage_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}

	msg := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
			Content: "hello", CreatedAt: time.Now(),
		},
		AuthorUsername: "testuser",
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.MessageWithAuthor, error) {
			if id == testMsgID {
				return msg, nil
			}
			return nil, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	err := h.GetMessage(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetMessage_WrongChannel(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}

	// Message belongs to a different channel.
	msg := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: 9999, AuthorID: testUserID,
			Content: "hello", CreatedAt: time.Now(),
		},
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return msg, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	_ = h.GetMessage(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// EditMessage tests
// ---------------------------------------------------------------------------

func TestEditMessage_AsAuthor(t *testing.T) {
	guilds := &mockGuildRepo{}
	members := &mockMemberRepo{}
	roles := &mockRoleRepo{}
	overrides := &mockChannelOverrideRepo{}
	channels := channelMock()
	gw := &mockGateway{}

	original := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
			Content: "original", CreatedAt: time.Now(),
		},
		AuthorUsername: "testuser",
	}
	edited := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
			Content: "edited", CreatedAt: time.Now(),
		},
		AuthorUsername: "testuser",
	}

	callCount := 0
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			callCount++
			if callCount == 1 {
				return original, nil
			}
			return edited, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPatch, "/api/v1/channels/2000/messages/5000", strings.NewReader(`{"content":"edited"}`))
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	err := h.EditMessage(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 1 || gw.events[0].Event != gateway.EventMessageUpdate {
		t.Fatalf("expected MESSAGE_UPDATE event, got %+v", gw.events)
	}
}

func TestEditMessage_NotAuthor(t *testing.T) {
	guilds := &mockGuildRepo{}
	members := &mockMemberRepo{}
	roles := &mockRoleRepo{}
	overrides := &mockChannelOverrideRepo{}
	channels := channelMock()
	gw := &mockGateway{}

	otherUserID := int64(7777)
	msg := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: otherUserID,
			Content: "someone else's message", CreatedAt: time.Now(),
		},
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return msg, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPatch, "/api/v1/channels/2000/messages/5000", strings.NewReader(`{"content":"hacked"}`))
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	_ = h.EditMessage(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DeleteMessage tests
// ---------------------------------------------------------------------------

func TestDeleteMessage_AsAuthor(t *testing.T) {
	guilds := &mockGuildRepo{}
	members := &mockMemberRepo{}
	roles := &mockRoleRepo{}
	overrides := &mockChannelOverrideRepo{}
	channels := channelMock()
	gw := &mockGateway{}

	msg := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
			Content: "my message", CreatedAt: time.Now(),
		},
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return msg, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/2000/messages/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	err := h.DeleteMessage(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 1 || gw.events[0].Event != gateway.EventMessageDelete {
		t.Fatalf("expected MESSAGE_DELETE event, got %+v", gw.events)
	}
}

func TestDeleteMessage_WithManageMessages(t *testing.T) {
	otherUserID := int64(7777)

	// Non-author but has ManageMessages permission.
	guilds, members, roles, overrides := permMocks(permissions.PermManageMessages | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}

	msg := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: otherUserID,
			Content: "other's message", CreatedAt: time.Now(),
		},
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return msg, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/2000/messages/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	err := h.DeleteMessage(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteMessage_NoPermission(t *testing.T) {
	otherUserID := int64(7777)

	// Non-author and no ManageMessages. Only ViewChannel.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}

	msg := &models.MessageWithAuthor{
		Message: models.Message{
			ID: testMsgID, ChannelID: testChannelID, AuthorID: otherUserID,
			Content: "other's message", CreatedAt: time.Now(),
		},
	}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return msg, nil
		},
	}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/2000/messages/5000", nil)
	c.SetParamNames("id", "message_id")
	c.SetParamValues("2000", "5000")
	setAuthUser(c, testUserID)

	_ = h.DeleteMessage(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Typing tests
// ---------------------------------------------------------------------------

func TestTyping_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermSendMessages | permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/typing", nil)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	err := h.Typing(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 1 || gw.events[0].Event != gateway.EventTypingStart {
		t.Fatalf("expected TYPING_START event, got %+v", gw.events)
	}
}

func TestTyping_NoPermission(t *testing.T) {
	// Has ViewChannel but NOT SendMessages.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{}

	h := newMessageHandler(msgs, channels, members, roles, guilds, overrides, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/typing", nil)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.Typing(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
