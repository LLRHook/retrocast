package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

// newTestChannelHandler creates a ChannelHandler backed by the given mock repos.
// When ownerID > 0, the guild mock returns that owner to allow permission checks.
func newTestChannelHandler(
	channels *mockChannelRepo,
	members *mockMemberRepo,
	guilds *mockGuildRepo,
	roles *mockRoleRepo,
	overrides *mockChannelOverrideRepo,
) *ChannelHandler {
	gw := &mockGateway{}
	sf := testSnowflake()
	perms := service.NewPermissionChecker(guilds, members, roles, overrides)
	svc := service.NewChannelService(channels, members, sf, gw, perms)
	return NewChannelHandler(svc)
}

// allowAllChannelHandler builds a handler where the caller is always the guild owner.
func allowAllChannelHandler(channels *mockChannelRepo, members *mockMemberRepo) *ChannelHandler {
	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			// Caller 1000 is always the owner.
			return &models.Guild{ID: id, OwnerID: 1000}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(_ context.Context, _, _ int64) ([]models.Role, error) {
			return []models.Role{{Permissions: int64(permissions.PermAdministrator)}}, nil
		},
		GetByGuildIDFn: func(_ context.Context, _ int64) ([]models.Role, error) {
			return []models.Role{{ID: 1, IsDefault: true, Permissions: int64(permissions.PermAdministrator)}}, nil
		},
	}
	return newTestChannelHandler(channels, members, guilds, roles, &mockChannelOverrideRepo{})
}

func TestCreateChannel_Success(t *testing.T) {
	channels := &mockChannelRepo{
		GetByGuildIDFn: func(_ context.Context, _ int64) ([]models.Channel, error) {
			return []models.Channel{}, nil
		},
	}

	h := allowAllChannelHandler(channels, &mockMemberRepo{})

	body := strings.NewReader(`{"name":"new-channel","type":0}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds/500/channels", body)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, 1000)

	if err := h.CreateChannel(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Channel `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.Name != "new-channel" {
		t.Errorf("expected channel name 'new-channel', got %q", resp.Data.Name)
	}
	if resp.Data.Type != models.ChannelTypeText {
		t.Errorf("expected channel type %d, got %d", models.ChannelTypeText, resp.Data.Type)
	}
}

func TestCreateChannel_NoPermission(t *testing.T) {
	guilds := &mockGuildRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: id, OwnerID: 9999}, nil // caller 1000 is NOT owner
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(_ context.Context, gID, uID int64) (*models.Member, error) {
			return &models.Member{GuildID: gID, UserID: uID}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(_ context.Context, _, _ int64) ([]models.Role, error) {
			return []models.Role{{Permissions: 0}}, nil // no permissions
		},
		GetByGuildIDFn: func(_ context.Context, _ int64) ([]models.Role, error) {
			return []models.Role{{ID: 1, IsDefault: true, Permissions: 0}}, nil
		},
	}

	h := newTestChannelHandler(&mockChannelRepo{}, members, guilds, roles, &mockChannelOverrideRepo{})

	body := strings.NewReader(`{"name":"new-channel","type":0}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds/500/channels", body)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, 1000)

	if err := h.CreateChannel(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestListChannels_AsMember(t *testing.T) {
	const guildID int64 = 500
	const userID int64 = 1000

	channels := &mockChannelRepo{
		GetByGuildIDFn: func(_ context.Context, gID int64) ([]models.Channel, error) {
			if gID == guildID {
				return []models.Channel{
					{ID: 1, GuildID: guildID, Name: "general", Type: models.ChannelTypeText},
					{ID: 2, GuildID: guildID, Name: "voice", Type: models.ChannelTypeVoice},
				}, nil
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

	h := allowAllChannelHandler(channels, members)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/500/channels", nil)
	c.SetParamNames("id")
	c.SetParamValues("500")
	setAuthUser(c, userID)

	if err := h.ListChannels(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []models.Channel `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 channels, got %d", len(resp.Data))
	}
}

func TestGetChannel_AsMember(t *testing.T) {
	const channelID int64 = 100
	const guildID int64 = 500
	const userID int64 = 1000

	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{ID: channelID, GuildID: guildID, Name: "general"}, nil
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

	h := allowAllChannelHandler(channels, members)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/100", nil)
	c.SetParamNames("id")
	c.SetParamValues("100")
	setAuthUser(c, userID)

	if err := h.GetChannel(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Channel `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.Name != "general" {
		t.Errorf("expected channel name 'general', got %q", resp.Data.Name)
	}
}

func TestUpdateChannel_WithPermission(t *testing.T) {
	const channelID int64 = 100
	const guildID int64 = 500

	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{ID: channelID, GuildID: guildID, Name: "old-name"}, nil
			}
			return nil, nil
		},
	}

	h := allowAllChannelHandler(channels, &mockMemberRepo{})

	body := strings.NewReader(`{"name":"new-name"}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/channels/100", body)
	c.SetParamNames("id")
	c.SetParamValues("100")
	setAuthUser(c, 1000)

	if err := h.UpdateChannel(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.Channel `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.Name != "new-name" {
		t.Errorf("expected channel name 'new-name', got %q", resp.Data.Name)
	}
}

func TestDeleteChannel_WithPermission(t *testing.T) {
	const channelID int64 = 100
	const guildID int64 = 500

	var deleted bool
	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			if id == channelID {
				return &models.Channel{ID: channelID, GuildID: guildID, Name: "to-delete"}, nil
			}
			return nil, nil
		},
		DeleteFn: func(_ context.Context, id int64) error {
			deleted = true
			return nil
		},
	}

	h := allowAllChannelHandler(channels, &mockMemberRepo{})

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/100", nil)
	c.SetParamNames("id")
	c.SetParamValues("100")
	setAuthUser(c, 1000)

	if err := h.DeleteChannel(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected channel delete to be called")
	}
}
