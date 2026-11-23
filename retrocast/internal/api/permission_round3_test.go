package api

import (
	"context"
	"fmt"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
	"testing"
)

func TestMemberRoleDeltaLeavesUnchangedRolesAlone(t *testing.T) {
	for _, op := range []string{"add", "remove", "noop"} {
		t.Run(op, func(t *testing.T) {
			g, m, r, o := permMocks(permissions.PermManageRoles | permissions.PermKickMembers)
			r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
				return []models.Role{{Position: 2, Permissions: int64(permissions.PermManageRoles | permissions.PermKickMembers)}}, nil
			}
			r.GetByGuildIDFn = func(context.Context, int64) ([]models.Role, error) {
				return []models.Role{{ID: 10, Position: 9, Permissions: int64(permissions.PermBanMembers)}, {ID: 20, Position: 1, Permissions: int64(permissions.PermKickMembers)}, {ID: 30, IsDefault: true, Permissions: int64(permissions.PermViewChannel)}}, nil
			}
			current := []int64{10, 30}
			requested := []int64{10, 20}
			if op == "remove" {
				current = []int64{10, 20, 30}
				requested = []int64{10}
			}
			if op == "noop" {
				requested = []int64{10}
			}
			member := &models.Member{GuildID: 1000, UserID: 400, Roles: current}
			m.GetByGuildAndUserFn = func(context.Context, int64, int64) (*models.Member, error) { return member, nil }
			added, removed := 0, 0
			m.AddRoleFn = func(_ context.Context, g, u, id int64) error {
				if id != 20 {
					t.Fatalf("unchanged role added: %d", id)
				}
				added++
				return nil
			}
			m.RemoveRoleFn = func(_ context.Context, g, u, id int64) error {
				if id != 20 {
					t.Fatalf("unchanged role removed: %d", id)
				}
				removed++
				return nil
			}
			svc := service.NewMemberService(m, g, r, &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
			if _, err := svc.UpdateMember(context.Background(), 1000, 300, 400, nil, &requested); err != nil {
				t.Fatal(err)
			}
			wantAdd, wantRemove := 0, 0
			if op == "add" {
				wantAdd = 1
			}
			if op == "remove" {
				wantRemove = 1
			}
			if added != wantAdd || removed != wantRemove {
				t.Fatalf("delta writes %d/%d want %d/%d", added, removed, wantAdd, wantRemove)
			}
		})
	}
}

func TestRefusedMemberPatchDoesNotWriteNickname(t *testing.T) {
	for _, reason := range []string{"grant", "hierarchy", "invalidRole", "allowed"} {
		t.Run(reason, func(t *testing.T) {
			g, m, r, o := permMocks(permissions.PermManageRoles | permissions.PermManageNicknames | permissions.PermKickMembers)
			r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
				return []models.Role{{Position: 2, Permissions: int64(permissions.PermManageRoles | permissions.PermManageNicknames | permissions.PermKickMembers)}}, nil
			}
			role := models.Role{ID: 10, Position: 1, Permissions: int64(permissions.PermBanMembers)}
			want := "MISSING_PERMISSIONS"
			if reason == "allowed" {
				role.Permissions = int64(permissions.PermKickMembers)
			}

			if reason == "hierarchy" {
				role.Position = 2
				role.Permissions = int64(permissions.PermKickMembers)
				want = "ROLE_HIERARCHY"
			}
			if reason == "invalidRole" {
				want = "INVALID_ROLE"
			}
			r.GetByGuildIDFn = func(context.Context, int64) ([]models.Role, error) { return []models.Role{role}, nil }
			old := "original"
			member := &models.Member{GuildID: 1000, UserID: 400, Roles: []int64{10}, Nickname: &old}
			m.GetByGuildAndUserFn = func(context.Context, int64, int64) (*models.Member, error) { return member, nil }
			writes := 0
			m.UpdateFn = func(context.Context, *models.Member) error { writes++; return nil }
			m.RemoveRoleFn = func(context.Context, int64, int64, int64) error { writes++; return nil }
			m.AddRoleFn = func(context.Context, int64, int64, int64) error { writes++; return nil }
			gw := &mockGateway{}
			svc := service.NewMemberService(m, g, r, gw, service.NewPermissionChecker(g, m, r, o))
			nick := "changed"
			ids := []int64{}
			if reason == "invalidRole" {
				ids = []int64{999}
			}
			_, err := svc.UpdateMember(context.Background(), 1000, 300, 400, &nick, &ids)
			if reason == "allowed" {
				if err != nil || writes != 2 || member.Nickname == nil || *member.Nickname != "changed" || len(gw.events) != 1 {
					t.Fatalf("allowed combined patch: %v, writes %d", err, writes)
				}
				return
			}

			se, ok := err.(*service.ServiceError)
			if !ok || se.Code != want {
				t.Fatalf("want %s got %v", want, err)
			}
			if writes != 0 || member.Nickname == nil || *member.Nickname != "original" || len(gw.events) != 0 {
				t.Fatal(fmt.Sprintf("refused patch changed state: writes %d nick %v events %d", writes, member.Nickname, len(gw.events)))
			}
		})
	}
}
