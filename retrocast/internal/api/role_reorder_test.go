package api

import (
	"context"
	"errors"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
	"net/http"
	"strings"
	"testing"
)

func TestRoleReorder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		actor int64
		order []string
		stale bool
		want  error
	}{
		{"no room", 200, []string{"2", "1", "4", "3"}, false, service.ErrConflict},
		{"tied", 200, []string{"2", "1", "3"}, false, nil},
		{"target highest", 200, []string{"1", "3", "2"}, false, service.ErrRoleHierarchy},
		{"grant", 200, []string{"2", "1", "3"}, false, service.ErrForbidden},
		{"owner", 100, []string{"2", "1", "3"}, false, nil},
		{"manager", 200, []string{"2", "1", "3"}, false, nil},
		{"higher role", 200, []string{"3", "2", "1"}, false, service.ErrRoleHierarchy},
		{"everyone", 100, []string{"0", "1", "2"}, false, service.ErrBadRequest},
		{"duplicate", 100, []string{"1", "1", "3"}, false, service.ErrBadRequest},
		{"no permission", 400, []string{"2", "1", "3"}, false, service.ErrForbidden},
		{"stale", 100, []string{"2", "1", "3"}, true, service.ErrConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := []models.Role{{ID: 0, GuildID: 1000, IsDefault: true}, {ID: 1, GuildID: 1000, Position: 1}, {ID: 2, GuildID: 1000, Position: 2}, {ID: 3, GuildID: 1000, Position: 3}}
			if tc.name == "no room" {
				current = append(current, models.Role{ID: 4, GuildID: 1000, Position: 1})
			}
			if tc.name == "tied" {
				current[2].Position = 1
			}
			if tc.name == "grant" {
				current[1].Permissions = int64(permissions.PermBanMembers)
			}
			g, m, r, o := permMocks(permissions.PermViewChannel)
			g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1000, OwnerID: 100}, nil }
			m.GetByGuildAndUserFn = func(context.Context, int64, int64) (*models.Member, error) {
				return &models.Member{GuildID: 1000, UserID: 200}, nil
			}
			r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
				if tc.actor == 400 {
					return nil, nil
				}
				return []models.Role{{ID: 3, Position: 3, Permissions: int64(permissions.PermManageRoles)}}, nil
			}
			r.GetByGuildIDFn = func(context.Context, int64) ([]models.Role, error) { return current, nil }
			writes := 0
			r.UpdateFn = func(_ context.Context, role *models.Role) error { writes++; return nil }
			svc := service.NewRoleService(g, r, m, channelMock(), o, testSnowflake(), &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
			expected := []service.RolePosition{}
			for _, role := range current {
				expected = append(expected, service.RolePosition{ID: role.ID, Position: role.Position})
			}
			if tc.stale {
				expected[1].Position = 9
			}
			_, err := svc.ReorderRoles(context.Background(), 1000, tc.actor, expected, tc.order)
			if tc.name == "no room" {
				var se *service.ServiceError
				if !errors.As(err, &se) || se.Code != "ROLE_ORDER_NO_ROOM" || !strings.Contains(se.Message, "owner") {
					t.Fatalf("want owner room message, got %v", err)
				}
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want != nil && writes != 0 {
				t.Fatal("refusal wrote roles")
			}
			wantWrites := 2
			if tc.name == "tied" {
				wantWrites = 1
			}
			if tc.want == nil && writes != wantWrites {
				t.Fatalf("writes %d", writes)
			}
		})
	}
}

func TestReorderRolesMissingBodyFields(t *testing.T) {
	for _, body := range []string{"", "{}", `{"expected":[]}`, `{"order":[]}`, `{"expected":null,"order":[]}`} {
		t.Run(body, func(t *testing.T) {
			guilds := &mockGuildRepo{GetByIDFn: func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1, OwnerID: 100}, nil }}
			roles := &mockRoleRepo{GetByGuildIDFn: func(context.Context, int64) ([]models.Role, error) {
				return []models.Role{{ID: 1, GuildID: 1, IsDefault: true}}, nil
			}}
			h := newRoleHandler(guilds, roles, &mockMemberRepo{}, &mockChannelRepo{}, &mockChannelOverrideRepo{}, &mockGateway{})
			c, rec := newTestContext(http.MethodPut, "/api/v1/guilds/1/roles/order", strings.NewReader(body))
			c.SetParamNames("id")
			c.SetParamValues("1")
			setAuthUser(c, 100)
			if err := h.ReorderRoles(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_ORDER") {
				t.Fatalf("want 400 INVALID_ORDER got %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}
