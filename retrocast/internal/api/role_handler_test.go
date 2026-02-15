package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
)

func newRoleHandler(
	guilds *mockGuildRepo,
	roles *mockRoleRepo,
	members *mockMemberRepo,
	channels *mockChannelRepo,
	overrides *mockChannelOverrideRepo,
	gw *mockGateway,
) *RoleHandler {
	return NewRoleHandler(guilds, roles, members, channels, overrides, testSnowflake(), gw)
}

func TestCreateRole_Success(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	roles := &mockRoleRepo{}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	body := `{"name":"Moderator","color":255,"permissions":"0","position":1}`
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds/1/roles", strings.NewReader(body))
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100) // owner

	err := h.CreateRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var role models.Role
	if err := json.Unmarshal(rec.Body.Bytes(), &role); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if role.Name != "Moderator" {
		t.Errorf("expected name Moderator, got %s", role.Name)
	}
	if role.Position != 1 {
		t.Errorf("expected position 1, got %d", role.Position)
	}
}

func TestCreateRole_HierarchyViolation(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 999}, nil // actor 100 is NOT the owner
		},
	}
	roles := &mockRoleRepo{
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			// Actor's highest role is position 5
			return []models.Role{{ID: 10, Position: 5}}, nil
		},
	}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	// Try to create role at position 5 (equal to highest) -> should fail
	body := `{"name":"HighRole","color":0,"permissions":"0","position":5}`
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds/1/roles", strings.NewReader(body))
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.CreateRole(c)
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
	if resp.Error.Code != "ROLE_HIERARCHY" {
		t.Errorf("expected ROLE_HIERARCHY, got %s", resp.Error.Code)
	}
}

func TestCreateRole_AsOwner_AnyPosition(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	roles := &mockRoleRepo{}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	// Owner creating a role at position 999 - should succeed
	body := `{"name":"TopRole","color":0,"permissions":"0","position":999}`
	c, rec := newTestContext(http.MethodPost, "/api/v1/guilds/1/roles", strings.NewReader(body))
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)

	err := h.CreateRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRole_Success(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Role, error) {
			return &models.Role{ID: 10, GuildID: 1, Name: "OldName", Position: 3}, nil
		},
	}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	body := `{"name":"NewName"}`
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/1/roles/10", strings.NewReader(body))
	c.SetParamNames("id", "role_id")
	c.SetParamValues("1", "10")
	setAuthUser(c, 100) // owner

	err := h.UpdateRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var role models.Role
	if err := json.Unmarshal(rec.Body.Bytes(), &role); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if role.Name != "NewName" {
		t.Errorf("expected NewName, got %s", role.Name)
	}
}

func TestUpdateRole_HierarchyViolation(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 999}, nil // 100 is not owner
		},
	}
	roles := &mockRoleRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Role, error) {
			return &models.Role{ID: 10, GuildID: 1, Name: "AdminRole", Position: 10}, nil
		},
		GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			return []models.Role{{ID: 20, Position: 5}}, nil // actor's highest is 5
		},
	}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	body := `{"name":"Renamed"}`
	c, rec := newTestContext(http.MethodPatch, "/api/v1/guilds/1/roles/10", strings.NewReader(body))
	c.SetParamNames("id", "role_id")
	c.SetParamValues("1", "10")
	setAuthUser(c, 100)

	err := h.UpdateRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteRole_Success(t *testing.T) {
	gw := &mockGateway{}
	deleted := false
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Role, error) {
			return &models.Role{ID: 10, GuildID: 1, Name: "Mods", Position: 3, IsDefault: false}, nil
		},
		DeleteFn: func(ctx context.Context, id int64) error {
			deleted = true
			return nil
		},
	}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/roles/10", nil)
	c.SetParamNames("id", "role_id")
	c.SetParamValues("1", "10")
	setAuthUser(c, 100)

	err := h.DeleteRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !deleted {
		t.Error("expected role to be deleted")
	}
}

func TestDeleteRole_EveryoneRole(t *testing.T) {
	gw := &mockGateway{}
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	roles := &mockRoleRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Role, error) {
			return &models.Role{ID: 10, GuildID: 1, Name: "@everyone", Position: 0, IsDefault: true}, nil
		},
	}
	h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/roles/10", nil)
	c.SetParamNames("id", "role_id")
	c.SetParamValues("1", "10")
	setAuthUser(c, 100)

	err := h.DeleteRole(c)
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
	if resp.Error.Code != "CANNOT_DELETE" {
		t.Errorf("expected CANNOT_DELETE, got %s", resp.Error.Code)
	}
}

func TestAssignRole_Success(t *testing.T) {
	gw := &mockGateway{}
	assigned := false
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		GetByGuildAndUserFn: func(ctx context.Context, guildID, userID int64) (*models.Member, error) {
			return &models.Member{GuildID: guildID, UserID: userID}, nil
		},
		AddRoleFn: func(ctx context.Context, guildID, userID, roleID int64) error {
			assigned = true
			return nil
		},
	}
	roles := &mockRoleRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Role, error) {
			return &models.Role{ID: 20, GuildID: 1, Name: "Mods", Position: 3}, nil
		},
	}
	h := newRoleHandler(guilds, roles, members, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/guilds/1/members/200/roles/20", nil)
	c.SetParamNames("id", "user_id", "role_id")
	c.SetParamValues("1", "200", "20")
	setAuthUser(c, 100) // owner

	err := h.AssignRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !assigned {
		t.Error("expected role to be assigned")
	}
}

func TestRemoveRole_Success(t *testing.T) {
	gw := &mockGateway{}
	removed := false
	guilds := &mockGuildRepo{
		GetByIDFn: func(ctx context.Context, id int64) (*models.Guild, error) {
			return &models.Guild{ID: 1, OwnerID: 100}, nil
		},
	}
	members := &mockMemberRepo{
		RemoveRoleFn: func(ctx context.Context, guildID, userID, roleID int64) error {
			removed = true
			return nil
		},
	}
	h := newRoleHandler(guilds, &mockRoleRepo{}, members, &mockChannelRepo{}, &mockChannelOverrideRepo{}, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/guilds/1/members/200/roles/20", nil)
	c.SetParamNames("id", "user_id", "role_id")
	c.SetParamValues("1", "200", "20")
	setAuthUser(c, 100) // owner

	err := h.RemoveRole(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !removed {
		t.Error("expected role to be removed")
	}
}
