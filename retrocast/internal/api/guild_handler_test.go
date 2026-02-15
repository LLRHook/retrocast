package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func newTestGuildHandler(
	guilds *mockGuildRepo,
	channels *mockChannelRepo,
	members *mockMemberRepo,
	roles *mockRoleRepo,
) *GuildHandler {
	gw := &mockGateway{}
	sf := testSnowflake()
	return NewGuildHandler(guilds, channels, members, roles, sf, gw)
}

func TestCreateGuild_Success(t *testing.T) {
	var guildCreated, rolesCreated, membersCreated, channelsCreated atomic.Int32

	guilds := &mockGuildRepo{
		CreateFn: func(_ context.Context, g *models.Guild) error {
			guildCreated.Add(1)
			return nil
		},
	}
	channels := &mockChannelRepo{
		CreateFn: func(_ context.Context, ch *models.Channel) error {
			channelsCreated.Add(1)
			return nil
		},
	}
	members := &mockMemberRepo{
		CreateFn: func(_ context.Context, m *models.Member) error {
			membersCreated.Add(1)
			return nil
		},
	}
	roles := &mockRoleRepo{
		CreateFn: func(_ context.Context, r *models.Role) error {
			rolesCreated.Add(1)
			return nil
		},
	}

	h := newTestGuildHandler(guilds, channels, members, roles)

	body := strings.NewReader(`{"name":"My Guild"}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds", body)
	setAuthUser(c, 1000)

	if err := h.CreateGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	// Verify guild was created.
	if guildCreated.Load() != 1 {
		t.Errorf("expected 1 guild created, got %d", guildCreated.Load())
	}

	// 2 roles: @everyone + Admin.
	if rolesCreated.Load() != 2 {
		t.Errorf("expected 2 roles created, got %d", rolesCreated.Load())
	}

	// 1 member (the owner).
	if membersCreated.Load() != 1 {
		t.Errorf("expected 1 member created, got %d", membersCreated.Load())
	}

	// 2 channels: #general text + General voice.
	if channelsCreated.Load() != 2 {
		t.Errorf("expected 2 channels created, got %d", channelsCreated.Load())
	}

	// Verify response body.
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected 'data' key in response")
	}
}

func TestGetGuild_AsMember(t *testing.T) {
	const guildID int64 = 500
	const userID int64 = 1000

	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			if id == guildID {
				return &models.Guild{ID: guildID, Name: "Test Guild", OwnerID: userID, CreatedAt: time.Now()}, nil
			}
			return nil, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(_ context.Context, gID, uID int64) (*models.Member, error) {
			if gID == guildID && uID == userID {
				return &models.Member{GuildID: guildID, UserID: userID}, nil
			}
			return nil, nil
		},
	}

	h := newTestGuildHandler(guilds, &mockChannelRepo{}, members, &mockRoleRepo{})

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/500", nil)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, userID)

	if err := h.GetGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestGetGuild_NotMember(t *testing.T) {
	guilds := &mockGuildRepo{}
	members := &mockMemberRepo{} // returns nil (not a member)

	h := newTestGuildHandler(guilds, &mockChannelRepo{}, members, &mockRoleRepo{})

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/500", nil)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, 9999)

	if err := h.GetGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestUpdateGuild_WithPermission(t *testing.T) {
	const guildID int64 = 500
	const ownerID int64 = 1000

	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			if id == guildID {
				return &models.Guild{ID: guildID, Name: "Old Name", OwnerID: ownerID}, nil
			}
			return nil, nil
		},
	}

	h := newTestGuildHandler(guilds, &mockChannelRepo{}, &mockMemberRepo{}, &mockRoleRepo{})

	body := strings.NewReader(`{"name":"New Name"}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/500", body)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, ownerID) // Owner has all permissions.

	if err := h.UpdateGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Guild `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.Name != "New Name" {
		t.Errorf("expected guild name 'New Name', got %q", resp.Data.Name)
	}
}

func TestUpdateGuild_WithoutPermission(t *testing.T) {
	const guildID int64 = 500
	const ownerID int64 = 1000
	const callerID int64 = 2000

	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			if id == guildID {
				return &models.Guild{ID: guildID, Name: "Guild", OwnerID: ownerID}, nil
			}
			return nil, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(_ context.Context, gID, uID int64) (*models.Member, error) {
			if gID == guildID && uID == callerID {
				return &models.Member{GuildID: guildID, UserID: callerID}, nil
			}
			return nil, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(_ context.Context, _, _ int64) ([]models.Role, error) {
			// No roles with MANAGE_GUILD permission.
			return []models.Role{{Permissions: 0}}, nil
		},
	}

	h := newTestGuildHandler(guilds, &mockChannelRepo{}, members, roles)

	body := strings.NewReader(`{"name":"Hacked"}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/500", body)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, callerID)

	if err := h.UpdateGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestDeleteGuild_AsOwner(t *testing.T) {
	const guildID int64 = 500
	const ownerID int64 = 1000

	var deleted bool
	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			if id == guildID {
				return &models.Guild{ID: guildID, Name: "Guild", OwnerID: ownerID}, nil
			}
			return nil, nil
		},
		DeleteFn: func(_ context.Context, id int64) error {
			deleted = true
			return nil
		},
	}

	h := newTestGuildHandler(guilds, &mockChannelRepo{}, &mockMemberRepo{}, &mockRoleRepo{})

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/500", nil)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, ownerID)

	if err := h.DeleteGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected guild delete to be called")
	}
}

func TestDeleteGuild_NotOwner(t *testing.T) {
	const guildID int64 = 500
	const ownerID int64 = 1000
	const callerID int64 = 2000

	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			if id == guildID {
				return &models.Guild{ID: guildID, Name: "Guild", OwnerID: ownerID}, nil
			}
			return nil, nil
		},
	}

	h := newTestGuildHandler(guilds, &mockChannelRepo{}, &mockMemberRepo{}, &mockRoleRepo{})

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/500", nil)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, callerID)

	if err := h.DeleteGuild(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}
