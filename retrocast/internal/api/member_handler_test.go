package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

func newMemberHandler(
	members *mockMemberRepo,
	guilds *mockGuildRepo,
	roles *mockRoleRepo,
	gw *mockGateway,
) *MemberHandler {
	perms := service.NewPermissionChecker(guilds, members, roles, &mockChannelOverrideRepo{})
	svc := service.NewMemberService(members, guilds, roles, gw, perms)
	return NewMemberHandler(svc)
}

// newMemberHandlerOwner creates a member handler where caller 100 is the guild owner.
func newMemberHandlerOwner(
	members *mockMemberRepo,
	roles *mockRoleRepo,
	gw *mockGateway,
) *MemberHandler {
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: id, OwnerID: 100}, nil
		},
	}
	return newMemberHandler(members, guilds, roles, gw)
}

func TestListMembers_AsMember(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil
		},
		GetByGuildIDFn: func(ctx context.Context, guildID int64, limit, offset int) ([]models.Member, error) {
			return []models.Member{
				{GuildID: 1, UserID: 100, JoinedAt: now},
				{GuildID: 1, UserID: 200, JoinedAt: now},
			}, nil
		},
	}
	h := newMemberHandlerOwner(members, &mockRoleRepo{}, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1/members", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.ListMembers(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	var memberList []models.Member
	if err := json.Unmarshal(resp["data"], &memberList); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if len(memberList) != 2 {
		t.Errorf("expected 2 members, got %d", len(memberList))
	}
}

func TestListMembers_NotMember(t *testing.T) {
	gw := &mockGateway{}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return nil, nil // not a member
		},
	}
	h := newMemberHandlerOwner(members, &mockRoleRepo{}, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1/members", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.ListMembers(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetMember_Success(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil
		},
	}
	h := newMemberHandlerOwner(members, &mockRoleRepo{}, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/guilds/1/members/200", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100)

	err := h.GetMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateSelf_Nickname(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	updated := false
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil
		},
		UpdateFn: func(ctx context.Context, member *models.Member) error {
			updated = true
			return nil
		},
	}
	h := newMemberHandlerOwner(members, &mockRoleRepo{}, gw)

	body := `{"nickname":"CoolNick"}`
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/1/members/@me", strings.NewReader(body))
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.UpdateSelf(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !updated {
		t.Error("expected member to be updated")
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	var member models.Member
	if err := json.Unmarshal(resp["data"], &member); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if member.Nickname == nil || *member.Nickname != "CoolNick" {
		t.Errorf("expected nickname CoolNick, got %v", member.Nickname)
	}
}

func TestUpdateMember_Nickname(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	updated := false
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil
		},
		UpdateFn: func(ctx context.Context, member *models.Member) error {
			updated = true
			return nil
		},
	}
	h := newMemberHandlerOwner(members, &mockRoleRepo{}, gw)

	body := `{"nickname":"NewNick"}`
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/1/members/200", strings.NewReader(body))
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100) // caller (owner)

	err := h.UpdateMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !updated {
		t.Error("expected member to be updated")
	}
}

func TestUpdateMember_Roles(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	addedRoles := []int64{}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now, Roles: []int64{}}, nil
		},
		AddRoleFn: func(ctx context.Context, guildID, userID, roleID int64) error {
			addedRoles = append(addedRoles, roleID)
			return nil
		},
		RemoveRoleFn: func(ctx context.Context, guildID, userID, roleID int64) error {
			return nil
		},
	}
	roles := &mockRoleRepo{
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Role, error) {
			return []models.Role{
				{ID: 1, GuildID: 1, IsDefault: true},
				{ID: 10, GuildID: 1, Name: "Mod", IsDefault: false},
				{ID: 20, GuildID: 1, Name: "VIP", IsDefault: false},
			}, nil
		},
	}
	h := newMemberHandlerOwner(members, roles, gw)

	body := `{"roles":[10,20]}`
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/1/members/200", strings.NewReader(body))
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100)

	err := h.UpdateMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(addedRoles) != 2 {
		t.Errorf("expected 2 roles added, got %d", len(addedRoles))
	}
}

func TestKickMember_Success(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	deleted := false
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil
		},
		DeleteFn: func(ctx context.Context, guildID, userID int64) error {
			deleted = true
			return nil
		},
	}
	h := newMemberHandler(members, guilds, &mockRoleRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/members/200", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100)

	err := h.KickMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected member to be deleted")
	}
}

func TestKickMember_Owner(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 200}, nil // target 200 is the owner
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: time.Now()}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			return []models.Role{{ID: 10, Position: 5, Permissions: int64(permissions.PermKickMembers)}}, nil
		},
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Role, error) {
			return []models.Role{{ID: 1, IsDefault: true, Permissions: 0}}, nil
		},
	}
	h := newMemberHandler(members, guilds, roles, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/members/200", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100)

	err := h.KickMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickMember_NoPermission(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 999}, nil // caller is not owner
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			return []models.Role{{Permissions: 0}}, nil // no permissions
		},
		GetByGuildIDFn: func(ctx context.Context, guildID int64) ([]models.Role, error) {
			return []models.Role{{ID: 1, IsDefault: true, Permissions: 0}}, nil
		},
	}
	h := newMemberHandler(members, guilds, roles, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/members/200", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("1", "200")
	setAuthUser(c, 100)

	err := h.KickMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLeaveGuild_Success(t *testing.T) {
	gw := &mockGateway{}
	now := time.Now()
	deleted := false
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 999}, nil // caller is NOT owner
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID, JoinedAt: now}, nil
		},
		DeleteFn: func(ctx context.Context, guildID, userID int64) error {
			deleted = true
			return nil
		},
	}
	h := newMemberHandler(members, guilds, &mockRoleRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/members/@me", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.LeaveGuild(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected member to be deleted")
	}
}

func TestLeaveGuild_AsOwner(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil // caller IS the owner
		},
	}
	h := newMemberHandler(&mockMemberRepo{}, guilds, &mockRoleRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/members/@me", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.LeaveGuild(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
