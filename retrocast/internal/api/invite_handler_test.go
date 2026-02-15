package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
)

func newInviteHandler(
	invites *mockInviteRepo,
	guilds *mockGuildRepo,
	members *mockMemberRepo,
	roles *mockRoleRepo,
	bans *mockBanRepo,
	gw *mockGateway,
) *InviteHandler {
	perms := service.NewPermissionChecker(guilds, members, roles, &mockChannelOverrideRepo{})
	svc := service.NewInviteService(invites, guilds, members, bans, gw, perms)
	return NewInviteHandler(svc)
}

func TestCreateInvite_Success(t *testing.T) {
	gw := &mockGateway{}
	created := false
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	invites := &mockInviteRepo{
		CreateFn: func(ctx context.Context, invite *models.Invite) error {
			created = true
			return nil
		},
	}
	// Owner bypasses permission checks, so no member/role mocking needed.
	h := newInviteHandler(invites, guilds, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, gw)

	body := `{"max_uses":10,"max_age_seconds":3600}`
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds/1/invites", strings.NewReader(body))
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.CreateInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if !created {
		t.Error("expected invite to be created")
	}

	var invite models.Invite
	if err := json.Unmarshal(rec.Body.Bytes(), &invite); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if invite.Code == "" {
		t.Error("expected non-empty invite code")
	}
	if invite.MaxUses != 10 {
		t.Errorf("expected max_uses 10, got %d", invite.MaxUses)
	}
}

func TestGetInvite_Public(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	future := now.Add(24 * time.Hour)
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, Name: "Test Guild", OwnerID: 100}, nil
		},
	}
	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "abc12345",
				GuildID:   1,
				CreatorID: 100,
				ExpiresAt: &future,
				CreatedAt: now,
			}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildIDFn: func(ctx context.Context, guildID int64, limit, offset int) ([]models.Member, error) {
			return []models.Member{
				{GuildID: 1, UserID: 100},
				{GuildID: 1, UserID: 200},
			}, nil
		},
	}
	h := newInviteHandler(invites, guilds, members, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/invites/abc12345", nil)
	c.SetParamNames("code")
	c.SetParamValues("abc12345")
	// No auth needed for GetInvite

	err := h.GetInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp service.InviteInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.GuildName != "Test Guild" {
		t.Errorf("expected guild name 'Test Guild', got %s", resp.GuildName)
	}
	if resp.MemberCount != 2 {
		t.Errorf("expected member count 2, got %d", resp.MemberCount)
	}
}

func TestGetInvite_Expired(t *testing.T) {
	gw := &mockGateway{}
	past := time.Now().Add(-1 * time.Hour)
	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "expired1",
				GuildID:   1,
				CreatorID: 100,
				ExpiresAt: &past,
			}, nil
		},
	}
	h := newInviteHandler(invites, &mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/invites/expired1", nil)
	c.SetParamNames("code")
	c.SetParamValues("expired1")

	err := h.GetInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAcceptInvite_Success(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	future := now.Add(24 * time.Hour)
	memberCreated := false
	usesIncremented := false

	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "valid123",
				GuildID:   1,
				CreatorID: 100,
				MaxUses:   10,
				Uses:      5,
				ExpiresAt: &future,
				CreatedAt: now,
			}, nil
		},
		IncrementUsesFn: func(ctx context.Context, code string) error {
			usesIncremented = true
			return nil
		},
	}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, Name: "Test Guild", OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return nil, nil // not yet a member
		},
		CreateFn: func(ctx context.Context, member *models.Member) error {
			memberCreated = true
			return nil
		},
	}
	h := newInviteHandler(invites, guilds, members, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/invites/valid123", nil)
	c.SetParamNames("code")
	c.SetParamValues("valid123")
	setAuthUser(c, 200) // new user joining

	err := h.AcceptInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !memberCreated {
		t.Error("expected member to be created")
	}
	if !usesIncremented {
		t.Error("expected uses to be incremented")
	}
}

func TestAcceptInvite_AlreadyMember(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	future := now.Add(24 * time.Hour)

	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "valid123",
				GuildID:   1,
				CreatorID: 100,
				ExpiresAt: &future,
				CreatedAt: now,
			}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil // already member
		},
	}
	h := newInviteHandler(invites, &mockGuildRepo{}, members, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/invites/valid123", nil)
	c.SetParamNames("code")
	c.SetParamValues("valid123")
	setAuthUser(c, 200)

	err := h.AcceptInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Error.Code != "ALREADY_MEMBER" {
		t.Errorf("expected ALREADY_MEMBER, got %s", resp.Error.Code)
	}
}

func TestAcceptInvite_MaxUsesReached(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	future := now.Add(24 * time.Hour)

	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "maxed123",
				GuildID:   1,
				CreatorID: 100,
				MaxUses:   5,
				Uses:      5, // already at max
				ExpiresAt: &future,
				CreatedAt: now,
			}, nil
		},
	}
	h := newInviteHandler(invites, &mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/invites/maxed123", nil)
	c.SetParamNames("code")
	c.SetParamValues("maxed123")
	setAuthUser(c, 200)

	err := h.AcceptInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Error.Code != "MAX_USES" {
		t.Errorf("expected MAX_USES, got %s", resp.Error.Code)
	}
}

func TestRevokeInvite_AsCreator(t *testing.T) {
	gw := &mockGateway{}
	deleted := false
	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "myinvite",
				GuildID:   1,
				CreatorID: 100, // same as caller
			}, nil
		},
		DeleteFn: func(ctx context.Context, code string) error {
			deleted = true
			return nil
		},
	}
	h := newInviteHandler(invites, &mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/invites/myinvite", nil)
	c.SetParamNames("code")
	c.SetParamValues("myinvite")
	setAuthUser(c, 100) // same as creator

	err := h.RevokeInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected invite to be deleted")
	}
}

func TestRevokeInvite_WithManageGuild(t *testing.T) {
	gw := &mockGateway{}
	deleted := false
	invites := &mockInviteRepo{
		GetByCodeFn: func(ctx context.Context, code string) (*models.Invite, error) {
			return &models.Invite{
				Code:      "other",
				GuildID:   1,
				CreatorID: 999, // different from caller
			}, nil
		},
		DeleteFn: func(ctx context.Context, code string) error {
			deleted = true
			return nil
		},
	}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil // caller is owner, bypasses permission check
		},
	}
	h := newInviteHandler(invites, guilds, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/invites/other", nil)
	c.SetParamNames("code")
	c.SetParamValues("other")
	setAuthUser(c, 100) // owner has MANAGE_GUILD implicitly

	err := h.RevokeInvite(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected invite to be deleted")
	}
}
