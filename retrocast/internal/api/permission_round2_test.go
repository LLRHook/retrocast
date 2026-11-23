package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
	"net/http"
	"testing"
)

func TestGrantLimits(t *testing.T) {
	for _, actor := range []int64{100, 200, 300} {
		for _, bits := range []permissions.Permission{permissions.PermViewChannel, permissions.PermKickMembers, permissions.PermAdministrator, permissions.PermManageGuild} {
			for _, op := range []string{"create", "update", "updateExisting", "everyone", "assign", "remove", "replaceAdd", "replaceRemove", "override"} {
				t.Run(fmt.Sprintf("%d/%d/%s", actor, bits, op), func(t *testing.T) {
					ctx := context.Background()
					g, m, r, o := permMocks(permissions.PermViewChannel)
					g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1000, OwnerID: 100}, nil }
					m.GetByGuildAndUserFn = func(_ context.Context, g, u int64) (*models.Member, error) {
						member := &models.Member{GuildID: g, UserID: u}
						if op == "replaceRemove" {
							member.Roles = []int64{10}
						}
						return member, nil
					}
					baseRole := models.Role{ID: 20, Position: 2, Permissions: int64(permissions.PermManageRoles | permissions.PermKickMembers)}
					r.GetByMemberFn = func(_ context.Context, g, u int64) ([]models.Role, error) {
						if u == 200 {
							return []models.Role{{ID: 30, Position: 2, Permissions: int64(permissions.PermAdministrator)}}, nil
						}
						return []models.Role{baseRole}, nil
					}
					target := models.Role{ID: 10, GuildID: 1000, Position: 1, Permissions: int64(bits)}
					if op == "everyone" {
						target.IsDefault = true
						target.Position = 0
					}
					r.GetByIDFn = func(context.Context, int64) (*models.Role, error) { copy := target; return &copy, nil }
					r.GetByGuildIDFn = func(context.Context, int64) ([]models.Role, error) {
						return []models.Role{{ID: testRoleID, IsDefault: true, Permissions: int64(permissions.PermViewChannel)}, target}, nil
					}
					writes := 0
					r.CreateFn = func(context.Context, *models.Role) error { writes++; return nil }
					r.UpdateFn = func(context.Context, *models.Role) error { writes++; return nil }
					m.AddRoleFn = func(context.Context, int64, int64, int64) error { writes++; return nil }
					m.RemoveRoleFn = func(context.Context, int64, int64, int64) error { writes++; return nil }
					o.SetFn = func(context.Context, *models.ChannelOverride) error { writes++; return nil }
					pc := service.NewPermissionChecker(g, m, r, o)
					svc := service.NewRoleService(g, r, m, channelMock(), o, testSnowflake(), &mockGateway{}, pc)
					var err error
					permBits := int64(bits)
					switch op {
					case "create":
						_, err = svc.CreateRole(ctx, 1000, actor, "role", 0, permBits, 1)
					case "update", "everyone":
						_, err = svc.UpdateRole(ctx, 1000, actor, 10, nil, nil, &permBits, nil)
					case "updateExisting":
						_, err = svc.UpdateRole(ctx, 1000, actor, 10, nil, nil, nil, nil)
					case "assign":
						err = svc.AssignRole(ctx, 1000, actor, 400, 10)
					case "remove":
						err = svc.RemoveRole(ctx, 1000, actor, 400, 10)
					case "override":
						_, err = svc.SetChannelOverride(ctx, 2000, actor, 10, permBits, 0)
					case "replaceAdd", "replaceRemove":
						ids := []int64{10}
						if op == "replaceRemove" {
							ids = nil
						}
						_, err = service.NewMemberService(m, g, r, &mockGateway{}, pc).UpdateMember(ctx, 1000, actor, 400, nil, &ids)
					}
					refused := actor == 300 && bits != permissions.PermKickMembers && bits != permissions.PermViewChannel
					if refused {
						se, ok := err.(*service.ServiceError)
						if !ok || se.Code != "MISSING_PERMISSIONS" {
							t.Fatalf("want MISSING_PERMISSIONS, got %v", err)
						}
						c, rec := newTestContext(http.MethodPost, "/", nil)
						_ = mapServiceError(c, err)
						if rec.Code != 403 {
							t.Fatalf("status %d", rec.Code)
						}
						if writes != 0 {
							t.Fatalf("refused operation wrote %d times", writes)
						}
					} else if err != nil || writes == 0 {
						t.Fatalf("allowed operation: err %v writes %d", err, writes)
					}
				})
			}
		}
	}
}

func TestEveryonePositionImmutable(t *testing.T) {
	for _, actor := range []int64{100, 200, 300} {
		for _, position := range []int{0, 1} {
			t.Run(fmt.Sprintf("%d/%d", actor, position), func(t *testing.T) {
				g, m, r, o := permMocks(permissions.PermManageRoles)
				g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{OwnerID: 100}, nil }
				r.GetByMemberFn = func(_ context.Context, g, u int64) ([]models.Role, error) {
					if u == 200 {
						return []models.Role{{Position: 5, Permissions: int64(permissions.PermAdministrator)}}, nil
					}
					return []models.Role{{Position: 5, Permissions: int64(permissions.PermManageRoles | permissions.PermKickMembers)}}, nil
				}
				r.GetByIDFn = func(context.Context, int64) (*models.Role, error) {
					return &models.Role{GuildID: 1000, IsDefault: true, Position: 0}, nil
				}
				writes := 0
				r.UpdateFn = func(context.Context, *models.Role) error { writes++; return nil }
				svc := service.NewRoleService(g, r, m, channelMock(), o, testSnowflake(), &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
				_, err := svc.UpdateRole(context.Background(), 1000, actor, 10, nil, nil, nil, &position)
				if position == 0 {
					if err != nil || writes != 1 {
						t.Fatalf("position 0: %v", err)
					}
				} else {
					se, ok := err.(*service.ServiceError)
					if !ok || se.Code != "INVALID_POSITION" || writes != 0 {
						t.Fatalf("position changed: %v", err)
					}
				}
			})
		}
	}
}

func TestCreateRoleRequiresManageRolesDespiteHighPosition(t *testing.T) {
	g, m, r, o := permMocks(permissions.PermKickMembers)
	r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
		return []models.Role{{Position: 5, Permissions: int64(permissions.PermKickMembers)}}, nil
	}
	svc := service.NewRoleService(g, r, m, channelMock(), o, testSnowflake(), &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
	_, err := svc.CreateRole(context.Background(), 1000, testUserID, "role", 0, int64(permissions.PermKickMembers), 1)
	se, ok := err.(*service.ServiceError)
	if !ok || se.Code != "MISSING_PERMISSIONS" {
		t.Fatalf("want missing permissions, got %v", err)
	}
	c, rec := newTestContext(http.MethodPost, "/", nil)
	_ = mapServiceError(c, err)
	if rec.Code != 403 {
		t.Fatalf("status %d", rec.Code)
	}
}

type fakeTypingStore struct {
	calls int
	err   error
}

func (f *fakeTypingStore) SetTyping(context.Context, int64, int64) error { f.calls++; return f.err }

func TestTypingMarkerAuthorization(t *testing.T) {
	for _, mode := range []string{"allowed", "refused", "storeFailure"} {
		t.Run(mode, func(t *testing.T) {
			bits := permissions.PermSendMessages
			if mode == "refused" {
				bits = 0
			}
			g, m, r, o := permMocks(bits)
			gw := &mockGateway{}
			h := newMessageHandler(&mockMessageRepo{}, channelMock(), m, r, g, o, gw)
			store := &fakeTypingStore{}
			if mode == "storeFailure" {
				store.err = errors.New("store unavailable")
			}
			h.service.SetTypingStore(store)
			err := h.service.Typing(context.Background(), testChannelID, testUserID)
			if mode == "allowed" {
				if err != nil || store.calls != 1 || len(gw.events) != 1 {
					t.Fatalf("allowed marker: %v, calls %d", err, store.calls)
				}
			} else if mode == "refused" {
				if err == nil || store.calls != 0 || len(gw.events) != 0 {
					t.Fatal("refused marker was stored or dispatched")
				}
			} else if err == nil || store.calls != 1 || len(gw.events) != 0 {
				t.Fatal("store failure was ignored")
			}
		})
	}
}

func TestKickGuildDisappeared(t *testing.T) {
	g, m, r, o := permMocks(permissions.PermKickMembers)
	calls := 0
	g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) {
		calls++
		if calls > 1 {
			return nil, nil
		}
		return &models.Guild{OwnerID: testOwnerID}, nil
	}
	err := service.NewMemberService(m, g, r, &mockGateway{}, service.NewPermissionChecker(g, m, r, o)).KickMember(context.Background(), 1000, testUserID, 400)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}
