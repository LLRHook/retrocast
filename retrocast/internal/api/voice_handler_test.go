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

const (
	testVoiceChannelID int64 = 7000
)

// newVoiceHandler wires up a VoiceHandler with the given mocks via the service layer.
func newVoiceHandler(
	voiceStates *mockVoiceStateRepo,
	channels *mockChannelRepo,
	users *mockUserRepo,
	gw *mockGateway,
	guilds *mockGuildRepo,
	members *mockMemberRepo,
	roles *mockRoleRepo,
	overrides *mockChannelOverrideRepo,
) *VoiceHandler {
	perms := service.NewPermissionChecker(guilds, members, roles, overrides)
	svc := service.NewVoiceService(voiceStates, channels, users, gw, perms, "test-api-key", "test-api-secret")
	return NewVoiceHandler(svc)
}

func voiceChannelMock() *mockChannelRepo {
	return &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			return &models.Channel{ID: testVoiceChannelID, GuildID: testGuildID, Name: "voice-chat", Type: models.ChannelTypeVoice}, nil
		},
	}
}

func voicePermMocks(everyonePerms permissions.Permission) (*mockGuildRepo, *mockMemberRepo, *mockRoleRepo, *mockChannelOverrideRepo) {
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
			return nil, nil
		},
		GetByGuildIDFn: func(_ context.Context, _ int64) ([]models.Role, error) {
			return []models.Role{
				{ID: testRoleID, GuildID: testGuildID, Name: "@everyone", Permissions: int64(everyonePerms), IsDefault: true},
			}, nil
		},
	}
	overrides := &mockChannelOverrideRepo{
		GetByChannelFn: func(_ context.Context, _ int64) ([]models.ChannelOverride, error) {
			return nil, nil
		},
	}
	return guilds, members, roles, overrides
}

// ---------------------------------------------------------------------------
// JoinVoice tests
// ---------------------------------------------------------------------------

func TestJoinVoice_Success(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermConnect | permissions.PermViewChannel)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			return &models.User{ID: testUserID, Username: "testuser"}, nil
		},
	}

	var upsertCalled bool
	voiceStates := &mockVoiceStateRepo{
		UpsertFn: func(_ context.Context, state *models.VoiceState) error {
			upsertCalled = true
			return nil
		},
		GetByChannelFn: func(_ context.Context, channelID int64) ([]models.VoiceState, error) {
			return []models.VoiceState{
				{GuildID: testGuildID, ChannelID: testVoiceChannelID, UserID: testUserID, SessionID: "voice-7000", JoinedAt: time.Now()},
			}, nil
		},
	}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/7000/voice/join", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	err := h.JoinVoice(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !upsertCalled {
		t.Fatal("expected voice state to be upserted")
	}

	var resp service.JoinChannelResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if len(resp.VoiceStates) != 1 {
		t.Fatalf("expected 1 voice state, got %d", len(resp.VoiceStates))
	}

	// Verify gateway event dispatched.
	if len(gw.events) == 0 {
		t.Fatal("expected gateway event to be dispatched")
	}
}

func TestJoinVoice_NotVoiceChannel(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermConnect | permissions.PermViewChannel)
	gw := &mockGateway{}

	// Return a text channel, not a voice channel.
	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			return &models.Channel{ID: testChannelID, GuildID: testGuildID, Name: "general", Type: models.ChannelTypeText}, nil
		},
	}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/voice/join", nil)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.JoinVoice(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestJoinVoice_NoPermission(t *testing.T) {
	// Has ViewChannel but NOT Connect.
	guilds, members, roles, overrides := voicePermMocks(permissions.PermViewChannel)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/7000/voice/join", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	_ = h.JoinVoice(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestJoinVoice_InvalidID(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermConnect | permissions.PermViewChannel)
	channels := voiceChannelMock()
	gw := &mockGateway{}
	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/abc/voice/join", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setAuthUser(c, testUserID)

	_ = h.JoinVoice(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestJoinVoice_ChannelNotFound(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermConnect | permissions.PermViewChannel)
	gw := &mockGateway{}

	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.Channel, error) {
			return nil, nil
		},
	}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/9999/voice/join", nil)
	c.SetParamNames("id")
	c.SetParamValues("9999")
	setAuthUser(c, testUserID)

	_ = h.JoinVoice(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// LeaveVoice tests
// ---------------------------------------------------------------------------

func TestLeaveVoice_Success(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermConnect | permissions.PermViewChannel)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{}
	var deleteCalled bool
	voiceStates := &mockVoiceStateRepo{
		GetByUserFn: func(_ context.Context, guildID, userID int64) (*models.VoiceState, error) {
			return &models.VoiceState{GuildID: testGuildID, ChannelID: testVoiceChannelID, UserID: testUserID, SessionID: "voice-7000", JoinedAt: time.Now()}, nil
		},
		DeleteFn: func(_ context.Context, guildID, userID int64) error {
			deleteCalled = true
			return nil
		},
	}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/7000/voice/leave", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	err := h.LeaveVoice(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !deleteCalled {
		t.Fatal("expected voice state to be deleted")
	}
	if len(gw.events) == 0 {
		t.Fatal("expected gateway event to be dispatched")
	}
}

func TestLeaveVoice_NotInChannel(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermConnect | permissions.PermViewChannel)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{
		GetByUserFn: func(_ context.Context, guildID, userID int64) (*models.VoiceState, error) {
			return nil, nil
		},
	}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/7000/voice/leave", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	_ = h.LeaveVoice(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetVoiceStates tests
// ---------------------------------------------------------------------------

func TestGetVoiceStates_Success(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermViewChannel | permissions.PermConnect)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{
		GetByChannelFn: func(_ context.Context, channelID int64) ([]models.VoiceState, error) {
			return []models.VoiceState{
				{GuildID: testGuildID, ChannelID: testVoiceChannelID, UserID: testUserID, SessionID: "voice-7000", JoinedAt: time.Now()},
			}, nil
		},
	}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/7000/voice/states", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	err := h.GetVoiceStates(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var states []models.VoiceState
	if err := json.Unmarshal(rec.Body.Bytes(), &states); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
}

func TestGetVoiceStates_NoPermission(t *testing.T) {
	// No ViewChannel permission.
	guilds, members, roles, overrides := voicePermMocks(permissions.PermSendMessages)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/7000/voice/states", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	_ = h.GetVoiceStates(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetVoiceStates_EmptyChannel(t *testing.T) {
	guilds, members, roles, overrides := voicePermMocks(permissions.PermViewChannel | permissions.PermConnect)
	channels := voiceChannelMock()
	gw := &mockGateway{}

	users := &mockUserRepo{}
	voiceStates := &mockVoiceStateRepo{
		GetByChannelFn: func(_ context.Context, channelID int64) ([]models.VoiceState, error) {
			return nil, nil
		},
	}

	h := newVoiceHandler(voiceStates, channels, users, gw, guilds, members, roles, overrides)

	c, rec := newTestContext(http.MethodGet, "/api/v1/channels/7000/voice/states", nil)
	c.SetParamNames("id")
	c.SetParamValues("7000")
	setAuthUser(c, testUserID)

	err := h.GetVoiceStates(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var states []models.VoiceState
	if err := json.Unmarshal(rec.Body.Bytes(), &states); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("expected 0 states, got %d", len(states))
	}
}
