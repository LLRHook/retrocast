package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

// ---------------------------------------------------------------------------
// Shared helpers for reaction tests
// ---------------------------------------------------------------------------

func newReactionHandler(
	reactions *mockReactionRepo,
	msgs *mockMessageRepo,
	chs *mockChannelRepo,
	guilds *mockGuildRepo,
	mems *mockMemberRepo,
	roles *mockRoleRepo,
	overrides *mockChannelOverrideRepo,
	gw *mockGateway,
) *ReactionHandler {
	perms := service.NewPermissionChecker(guilds, mems, roles, overrides)
	svc := service.NewReactionService(reactions, msgs, chs, &mockDMChannelRepo{}, gw, perms)
	return NewReactionHandler(svc)
}

func messageMock() *mockMessageRepo {
	return &mockMessageRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.MessageWithAuthor, error) {
			if id == testMsgID {
				return &models.MessageWithAuthor{
					Message: models.Message{
						ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
						Content: "hello", CreatedAt: time.Now(),
					},
					AuthorUsername: "testuser",
				}, nil
			}
			return nil, nil
		},
	}
}

// ---------------------------------------------------------------------------
// AddReaction tests
// ---------------------------------------------------------------------------

func TestAddReaction_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D/@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	err := h.AddReaction(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 1 || gw.events[0].Event != gateway.EventMessageReactionAdd {
		t.Fatalf("expected MESSAGE_REACTION_ADD event, got %+v", gw.events)
	}
}

func TestAddReaction_EmptyEmoji(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/messages/5000/reactions//@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "")
	setAuthUser(c, testUserID)

	_ = h.AddReaction(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddReaction_NoPermission(t *testing.T) {
	// Has ViewChannel but NOT ReadMessageHistory.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D/@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	_ = h.AddReaction(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddReaction_MessageNotFound(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return nil, nil
		},
	}
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/messages/9999/reactions/%F0%9F%91%8D/@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "9999", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	_ = h.AddReaction(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddReaction_MessageWrongChannel(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := &mockMessageRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.MessageWithAuthor, error) {
			return &models.MessageWithAuthor{
				Message: models.Message{
					ID: testMsgID, ChannelID: 9999, AuthorID: testUserID,
					Content: "hello", CreatedAt: time.Now(),
				},
			}, nil
		},
	}
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D/@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	_ = h.AddReaction(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// RemoveReaction tests
// ---------------------------------------------------------------------------

func TestRemoveReaction_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D/@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	err := h.RemoveReaction(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(gw.events) != 1 || gw.events[0].Event != gateway.EventMessageReactionRemove {
		t.Fatalf("expected MESSAGE_REACTION_REMOVE event, got %+v", gw.events)
	}
}

func TestRemoveReaction_NoPermission(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D/@me", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	_ = h.RemoveReaction(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetReactions tests
// ---------------------------------------------------------------------------

func TestGetReactions_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{
		GetUsersByReactionFn: func(_ context.Context, _ int64, _ string, _ int) ([]int64, error) {
			return []int64{testUserID, 7777}, nil
		},
	}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	err := h.GetReactions(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result []int64
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 user IDs, got %d", len(result))
	}
}

func TestGetReactions_InvalidLimit(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D?limit=0", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	c.QueryParams().Set("limit", "0")
	setAuthUser(c, testUserID)

	_ = h.GetReactions(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetReactions_NoPermission(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	gw := &mockGateway{}
	msgs := messageMock()
	reactions := &mockReactionRepo{}

	h := newReactionHandler(reactions, msgs, channels, guilds, members, roles, overrides, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/2000/messages/5000/reactions/%F0%9F%91%8D", nil)
	c.SetParamNames("id", "message_id", "emoji")
	c.SetParamValues("2000", "5000", "%F0%9F%91%8D")
	setAuthUser(c, testUserID)

	_ = h.GetReactions(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
