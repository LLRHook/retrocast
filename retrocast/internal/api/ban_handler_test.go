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
	"github.com/victorivanov/retrocast/internal/service"
)

func newBanHandler(
	guilds *mockGuildRepo,
	members *mockMemberRepo,
	roles *mockRoleRepo,
	bans *mockBanRepo,
	gw *mockGateway,
) *BanHandler {
	perms := service.NewPermissionChecker(guilds, members, roles, &mockChannelOverrideRepo{})
	svc := service.NewBanService(guilds, members, roles, bans, gw, perms)
	return NewBanHandler(svc)
}

func TestBanMember(t *testing.T) {
	gw := &mockGateway{}
	banCreated := false
	memberDeleted := false

	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		DeleteFn: func(ctx context.Context, guildID, userID int64) error {
			memberDeleted = true
			return nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			if userID == 200 {
				return []models.Role{{ID: 10, Position: 0}}, nil
			}
			return nil, nil
		},
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Role, error) {
			return []models.Role{
				{ID: 1, Name: "@everyone", Position: 0, IsDefault: true, Permissions: 0},
			}, nil
		},
	}
	bans := &mockBanRepo{
		CreateFn: func(ctx context.Context, ban *models.Ban) error {
			banCreated = true
			return nil
		},
	}

	h := newBanHandler(guilds, members, roles, bans, gw)

	body := `{"reason":"spam"}`
	c, rec := newTestContext(http.MethodPut, "/api/v1/guilds/1/bans/200", strings.NewReader(body))
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100) // owner

	err := h.BanMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !banCreated {
		t.Error("expected ban to be created")
	}
	if !memberDeleted {
		t.Error("expected member to be auto-kicked")
	}

	// Verify gateway events.
	gw.mu.Lock()
	defer gw.mu.Unlock()
	if len(gw.events) < 2 {
		t.Fatalf("expected at least 2 gateway events, got %d", len(gw.events))
	}
	if gw.events[0].Event != gateway.EventGuildBanAdd {
		t.Errorf("expected first event GUILD_BAN_ADD, got %s", gw.events[0].Event)
	}
	if gw.events[1].Event != gateway.EventGuildMemberRemove {
		t.Errorf("expected second event GUILD_MEMBER_REMOVE, got %s", gw.events[1].Event)
	}
}

func TestBanMember_CannotBanOwner(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: time.Now()}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			return []models.Role{{ID: 10, Position: 5, Permissions: int64(1 << 6)}}, nil // PermBanMembers
		},
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Role, error) {
			return []models.Role{
				{ID: 1, Name: "@everyone", Position: 0, IsDefault: true, Permissions: 0},
			}, nil
		},
	}

	h := newBanHandler(guilds, members, roles, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/guilds/1/bans/100", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "100") // trying to ban the owner
	setAuthUser(c, 300)          // non-owner with ban perms

	err := h.BanMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Error.Code != "FORBIDDEN" {
		t.Errorf("expected FORBIDDEN, got %s", resp.Error.Code)
	}
}

func TestBanMember_CannotBanSelf(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}

	h := newBanHandler(guilds, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/guilds/1/bans/100", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "100")
	setAuthUser(c, 100) // trying to ban self

	err := h.BanMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Error.Code != "CANNOT_BAN_SELF" {
		t.Errorf("expected CANNOT_BAN_SELF, got %s", resp.Error.Code)
	}
}

func TestBanMember_MissingPermission(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: time.Now()}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			return []models.Role{{ID: 10, Position: 1, Permissions: 0}}, nil // no permissions
		},
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Role, error) {
			return []models.Role{
				{ID: 1, Name: "@everyone", Position: 0, IsDefault: true, Permissions: 0},
			}, nil
		},
	}

	h := newBanHandler(guilds, members, roles, &mockBanRepo{}, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/guilds/1/bans/200", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 300)

	err := h.BanMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Error.Code != "MISSING_PERMISSIONS" {
		t.Errorf("expected MISSING_PERMISSIONS, got %s", resp.Error.Code)
	}
}

func TestUnbanMember(t *testing.T) {
	gw := &mockGateway{}
	banDeleted := false

	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	bans := &mockBanRepo{
		DeleteFn: func(ctx context.Context, guildID, userID int64) error {
			banDeleted = true
			return nil
		},
	}

	h := newBanHandler(guilds, &mockMemberRepo{}, &mockRoleRepo{}, bans, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/bans/200", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100) // owner

	err := h.UnbanMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !banDeleted {
		t.Error("expected ban to be deleted")
	}

	// Verify gateway event.
	gw.mu.Lock()
	defer gw.mu.Unlock()
	if len(gw.events) != 1 {
		t.Fatalf("expected 1 gateway event, got %d", len(gw.events))
	}
	if gw.events[0].Event != gateway.EventGuildBanRemove {
		t.Errorf("expected GUILD_BAN_REMOVE, got %s", gw.events[0].Event)
	}
}

func TestListBans(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()

	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	reason := "spam"
	bans := &mockBanRepo{
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Ban, error) {
			return []models.Ban{
				{GuildID: 1, UserID: 200, Reason: &reason, CreatedBy: 100, CreatedAt: now},
				{GuildID: 1, UserID: 300, CreatedBy: 100, CreatedAt: now},
			}, nil
		},
	}

	h := newBanHandler(guilds, &mockMemberRepo{}, &mockRoleRepo{}, bans, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1/bans", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100) // owner

	err := h.ListBans(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var banList []models.Ban
	if err := json.Unmarshal(rec.Body.Bytes(), &banList); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(banList) != 2 {
		t.Errorf("expected 2 bans, got %d", len(banList))
	}
}
