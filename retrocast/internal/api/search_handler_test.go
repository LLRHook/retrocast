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

// newSearchHandler wires up a SearchHandler with the given mocks via the service layer.
func newSearchHandler(
	msgs *mockMessageRepo,
	mems *mockMemberRepo,
	guilds *mockGuildRepo,
	roles *mockRoleRepo,
	overrides *mockChannelOverrideRepo,
) *SearchHandler {
	perms := service.NewPermissionChecker(guilds, mems, roles, overrides)
	svc := service.NewSearchService(msgs, mems, perms)
	return NewSearchHandler(svc)
}

// ---------------------------------------------------------------------------
// SearchMessages tests
// ---------------------------------------------------------------------------

func TestSearchMessages_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)

	results := []models.MessageWithAuthor{
		{
			Message: models.Message{
				ID: testMsgID, ChannelID: testChannelID, AuthorID: testUserID,
				Content: "hello world", CreatedAt: time.Now(),
			},
			AuthorUsername: "testuser",
		},
	}

	var capturedQuery string
	var capturedLimit int
	msgs := &mockMessageRepo{
		SearchMessagesFn: func(_ context.Context, _ int64, query string, _ *int64, _ *time.Time, _ *time.Time, limit int) ([]models.MessageWithAuthor, error) {
			capturedQuery = query
			capturedLimit = limit
			return results, nil
		},
	}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search?q=hello&limit=10", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	c.QueryParams().Set("q", "hello")
	c.QueryParams().Set("limit", "10")
	setAuthUser(c, testUserID)

	err := h.SearchMessages(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedQuery != "hello" {
		t.Fatalf("expected query 'hello', got %q", capturedQuery)
	}
	if capturedLimit != 10 {
		t.Fatalf("expected limit 10, got %d", capturedLimit)
	}

	var result []models.MessageWithAuthor
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestSearchMessages_DefaultLimit(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)

	var capturedLimit int
	msgs := &mockMessageRepo{
		SearchMessagesFn: func(_ context.Context, _ int64, _ string, _ *int64, _ *time.Time, _ *time.Time, limit int) ([]models.MessageWithAuthor, error) {
			capturedLimit = limit
			return nil, nil
		},
	}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search?q=test", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	c.QueryParams().Set("q", "test")
	setAuthUser(c, testUserID)

	err := h.SearchMessages(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedLimit != 25 {
		t.Fatalf("expected default limit 25, got %d", capturedLimit)
	}
}

func TestSearchMessages_EmptyQuery(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)
	msgs := &mockMessageRepo{}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	setAuthUser(c, testUserID)

	_ = h.SearchMessages(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchMessages_InvalidLimit(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)
	msgs := &mockMessageRepo{}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search?q=test&limit=200", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	c.QueryParams().Set("q", "test")
	c.QueryParams().Set("limit", "200")
	setAuthUser(c, testUserID)

	_ = h.SearchMessages(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchMessages_NoPermission(t *testing.T) {
	// Has ViewChannel but NOT ReadMessageHistory.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	msgs := &mockMessageRepo{}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search?q=hello", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	c.QueryParams().Set("q", "hello")
	setAuthUser(c, testUserID)

	_ = h.SearchMessages(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchMessages_NotMember(t *testing.T) {
	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: testGuildID, OwnerID: testOwnerID}, nil
		},
	}
	// Member lookup returns nil (not a member).
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(_ context.Context, _, _ int64) (*models.Member, error) {
			return nil, nil
		},
	}
	roles := &mockRoleRepo{}
	overrides := &mockChannelOverrideRepo{}
	msgs := &mockMessageRepo{}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search?q=hello", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	c.QueryParams().Set("q", "hello")
	setAuthUser(c, testUserID)

	_ = h.SearchMessages(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchMessages_WithAuthorFilter(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)

	var capturedAuthorID *int64
	msgs := &mockMessageRepo{
		SearchMessagesFn: func(_ context.Context, _ int64, _ string, authorID *int64, _ *time.Time, _ *time.Time, _ int) ([]models.MessageWithAuthor, error) {
			capturedAuthorID = authorID
			return nil, nil
		},
	}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1000/messages/search?q=hello&author_id=3000", nil)
	c.SetParamNames("id")
	c.SetParamValues("1000")
	c.QueryParams().Set("q", "hello")
	c.QueryParams().Set("author_id", "3000")
	setAuthUser(c, testUserID)

	err := h.SearchMessages(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedAuthorID == nil || *capturedAuthorID != testUserID {
		t.Fatalf("expected author_id %d, got %v", testUserID, capturedAuthorID)
	}
}

func TestSearchMessages_InvalidGuildID(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermReadMessageHistory | permissions.PermViewChannel)
	msgs := &mockMessageRepo{}

	h := newSearchHandler(msgs, members, guilds, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/abc/messages/search?q=hello", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	c.QueryParams().Set("q", "hello")
	setAuthUser(c, testUserID)

	_ = h.SearchMessages(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
