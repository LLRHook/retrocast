package api

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPermissionRegressionRoleOperations(t *testing.T) {
	for _, actor := range []int64{100, 200, 300, 400, 500} {
		for _, op := range []string{"create", "update", "delete", "assign", "remove", "override", "deleteOverride", "list"} {
			t.Run(fmt.Sprintf("%s/%d", op, actor), func(t *testing.T) {
				guilds := &mockGuildRepo{GetByIDFn: func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1, OwnerID: 100}, nil }}
				members := &mockMemberRepo{GetByGuildAndUserFn: func(_ context.Context, g, u int64) (*models.Member, error) {
					if u == 500 {
						return nil, nil
					}
					return &models.Member{GuildID: g, UserID: u}, nil
				}}
				roles := &mockRoleRepo{GetByIDFn: func(context.Context, int64) (*models.Role, error) {
					return &models.Role{ID: 10, GuildID: 1, Position: -1}, nil
				}, GetByMemberFn: func(_ context.Context, g, u int64) ([]models.Role, error) {
					switch u {
					case 200:
						return []models.Role{{Position: 5, Permissions: int64(permissions.PermAdministrator)}}, nil
					case 300:
						return []models.Role{{Position: 5, Permissions: int64(permissions.PermManageRoles)}}, nil
					}
					return nil, nil
				}}
				channels := &mockChannelRepo{GetByIDFn: func(context.Context, int64) (*models.Channel, error) { return &models.Channel{ID: 20, GuildID: 1}, nil }}
				h := newRoleHandler(guilds, roles, members, channels, &mockChannelOverrideRepo{}, &mockGateway{})
				body := `{"name":"new","allow":"0","deny":"0"}`
				if op == "create" {
					body = `{"name":"new","position":0}`
				}
				c, rec := newTestContext(http.MethodPost, "/", strings.NewReader(body))
				c.SetParamNames("id", "role_id", "user_id")
				c.SetParamValues("1", "10", "400")
				setAuthUser(c, actor)
				var err error
				switch op {
				case "create":
					err = h.CreateRole(c)
				case "update":
					err = h.UpdateRole(c)
				case "delete":
					err = h.DeleteRole(c)
				case "assign":
					err = h.AssignRole(c)
				case "remove":
					err = h.RemoveRole(c)
				case "override":
					err = h.SetChannelOverride(c)
				case "deleteOverride":
					err = h.DeleteChannelOverride(c)
				case "list":
					err = h.ListRoles(c)
				}
				if err != nil {
					t.Fatal(err)
				}
				if op == "list" && actor == 500 && rec.Code != 404 {
					t.Fatalf("non-member list = %d", rec.Code)
				}
				if op == "update" && actor == 400 && rec.Code != 403 {
					t.Fatalf("unprivileged update = %d", rec.Code)
				}
				allowed := actor <= 300 || (op == "list" && actor == 400)
				if allowed && rec.Code >= 400 {
					t.Fatalf("allowed actor %d: %d %s", actor, rec.Code, rec.Body.String())
				}
				if !allowed && rec.Code < 400 {
					t.Fatalf("refused actor %d: %d", actor, rec.Code)
				}
			})
		}
	}
}

func TestPermissionRegressionGuildEveryone(t *testing.T) {
	guilds := &mockGuildRepo{GetByIDFn: func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1, OwnerID: 100}, nil }}
	members := &mockMemberRepo{GetByGuildAndUserFn: func(_ context.Context, g, u int64) (*models.Member, error) {
		if u == 500 {
			return nil, nil
		}
		return &models.Member{GuildID: g, UserID: u}, nil
	}}
	roles := &mockRoleRepo{GetByGuildIDFn: func(context.Context, int64) ([]models.Role, error) {
		return []models.Role{{ID: 1, IsDefault: true, Permissions: int64(permissions.PermCreateInvite)}}, nil
	}}
	p := service.NewPermissionChecker(guilds, members, roles, &mockChannelOverrideRepo{})
	for _, u := range []int64{100, 400, 500} {
		err := p.RequireGuildPermission(context.Background(), 1, u, int64(permissions.PermCreateInvite))
		if (err == nil) != (u != 500) {
			t.Fatalf("actor %d: %v", u, err)
		}
	}
}

func TestPermissionRegressionRoleBoundaries(t *testing.T) {
	for _, op := range []string{"create", "update", "delete", "assign", "remove", "override", "deleteOverride"} {
		for _, boundary := range []string{"allowed", "foreign", "equal", "negative"} {
			t.Run(op+"/"+boundary, func(t *testing.T) {
				g, m, r, o := permMocks(permissions.PermManageRoles)
				r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) { return []models.Role{{Position: 5}}, nil }
				r.GetByIDFn = func(context.Context, int64) (*models.Role, error) {
					guild := testGuildID
					if boundary == "foreign" {
						guild++
					}
					return &models.Role{ID: 10, GuildID: guild, Position: 1}, nil
				}
				h := newRoleHandler(g, r, m, channelMock(), o, &mockGateway{})
				position := 1
				if boundary == "equal" {
					position = 5
				}
				if boundary == "negative" {
					position = -1
				}
				body := fmt.Sprintf(`{"name":"role","position":%d,"allow":"0","deny":"0"}`, position)
				c, rec := newTestContext(http.MethodPost, "/", strings.NewReader(body))
				c.SetParamNames("id", "role_id", "user_id")
				c.SetParamValues("1000", "10", "3000")
				setAuthUser(c, testUserID)
				switch op {
				case "create":
					_ = h.CreateRole(c)
				case "update":
					_ = h.UpdateRole(c)
				case "delete":
					_ = h.DeleteRole(c)
				case "assign":
					_ = h.AssignRole(c)
				case "remove":
					_ = h.RemoveRole(c)
				case "override":
					_ = h.SetChannelOverride(c)
				case "deleteOverride":
					_ = h.DeleteChannelOverride(c)
				}
				if op == "update" && boundary == "foreign" && rec.Code != 404 {
					t.Fatalf("foreign role update = %d", rec.Code)
				}
				refused := (boundary == "foreign" && op != "create") || ((boundary == "equal" || boundary == "negative") && (op == "create" || op == "update"))
				if (rec.Code >= 400) != refused {
					t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func TestPermissionRegressionSearchOverrides(t *testing.T) {
	for _, actor := range []int64{100, 200, 300, 400, 500} {
		t.Run(fmt.Sprint(actor), func(t *testing.T) {
			g, m, r, o := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
			g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1000, OwnerID: 100}, nil }
			m.GetByGuildAndUserFn = func(_ context.Context, g, u int64) (*models.Member, error) {
				if u == 500 {
					return nil, nil
				}
				return &models.Member{UserID: u, GuildID: g}, nil
			}
			r.GetByMemberFn = func(_ context.Context, g, u int64) ([]models.Role, error) {
				if u == 200 {
					return []models.Role{{Permissions: int64(permissions.PermAdministrator)}}, nil
				}
				return nil, nil
			}
			o.GetByChannelFn = func(_ context.Context, ch int64) ([]models.ChannelOverride, error) {
				if ch == 2 {
					return []models.ChannelOverride{{RoleID: testRoleID, Deny: int64(permissions.PermViewChannel)}}, nil
				}
				if ch == 3 {
					return []models.ChannelOverride{{RoleID: testRoleID, Deny: int64(permissions.PermReadMessageHistory)}}, nil
				}
				return nil, nil
			}
			msgs := &mockMessageRepo{SearchMessagesFn: func(context.Context, int64, string, *int64, *time.Time, *time.Time, int) ([]models.MessageWithAuthor, error) {
				return []models.MessageWithAuthor{{Message: models.Message{ChannelID: 1}}, {Message: models.Message{ChannelID: 2}}, {Message: models.Message{ChannelID: 3}}}, nil
			}}
			svc := service.NewSearchService(msgs, m, service.NewPermissionChecker(g, m, r, o))
			got, err := svc.SearchMessages(context.Background(), 1000, actor, "test", nil, nil, nil, 25)
			if actor == 500 {
				if err == nil {
					t.Fatal("outsider allowed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := 1
			if actor <= 200 {
				want = 3
			}
			if len(got) != want {
				t.Fatalf("got %d, want %d", len(got), want)
			}
		})
	}
}

func TestPermissionRegressionKick(t *testing.T) {
	for _, tc := range []struct {
		name          string
		actor, target int64
		position      int
		code          string
	}{{"owner", 100, 400, 0, ""}, {"administrator", 200, 400, 0, ""}, {"moderator", 300, 400, 0, ""}, {"equal", 300, 400, 5, "ROLE_HIERARCHY"}, {"higher", 300, 400, 6, "ROLE_HIERARCHY"}, {"self", 300, 300, 0, "CANNOT_KICK_SELF"}, {"owner target", 300, 100, 0, "FORBIDDEN"}, {"plain", 400, 300, 0, "MISSING_PERMISSIONS"}, {"outsider", 500, 300, 0, "FORBIDDEN"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, m, r, o := permMocks(0)
			g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{OwnerID: 100}, nil }
			m.GetByGuildAndUserFn = func(_ context.Context, g, u int64) (*models.Member, error) {
				if u == 500 {
					return nil, nil
				}
				return &models.Member{UserID: u, GuildID: g}, nil
			}
			r.GetByMemberFn = func(_ context.Context, g, u int64) ([]models.Role, error) {
				if u == 200 {
					return []models.Role{{Position: 5, Permissions: int64(permissions.PermAdministrator)}}, nil
				}
				if u == 300 {
					return []models.Role{{Position: 5, Permissions: int64(permissions.PermKickMembers)}}, nil
				}
				return []models.Role{{Position: tc.position}}, nil
			}
			svc := service.NewMemberService(m, g, r, &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
			err := svc.KickMember(context.Background(), 1000, tc.actor, tc.target)
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				e, ok := err.(*service.ServiceError)
				if !ok || e.Code != tc.code {
					t.Fatalf("got %v, want %s", err, tc.code)
				}
			}
		})
	}
}

func TestPermissionRegressionGuildDefaults(t *testing.T) {
	g, m, r, o := permMocks(0)
	r.CreateFn = func(_ context.Context, role *models.Role) error {
		if role.IsDefault && role.Permissions != int64(permissions.DefaultEveryonePerms) {
			t.Errorf("default permissions %d", role.Permissions)
		}
		return nil
	}
	svc := service.NewGuildService(g, &mockChannelRepo{}, m, r, testSnowflake(), &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
	if _, err := svc.CreateGuild(context.Background(), 100, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestPermissionRegressionTyping(t *testing.T) {
	for _, kind := range []string{"guild", "dm", "unknown"} {
		for _, actor := range []int64{100, 200, 300, 400, 500} {
			t.Run(fmt.Sprintf("%s/%d", kind, actor), func(t *testing.T) {
				g, m, r, o := permMocks(permissions.PermSendMessages)
				g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{OwnerID: 100}, nil }
				m.GetByGuildAndUserFn = func(_ context.Context, g, u int64) (*models.Member, error) {
					if u == 500 {
						return nil, nil
					}
					return &models.Member{GuildID: g, UserID: u}, nil
				}
				r.GetByMemberFn = func(_ context.Context, g, u int64) ([]models.Role, error) {
					if u == 200 {
						return []models.Role{{Permissions: int64(permissions.PermAdministrator)}}, nil
					}
					if u == 300 {
						return []models.Role{{ID: 10}}, nil
					}
					return nil, nil
				}
				o.GetByChannelFn = func(context.Context, int64) ([]models.ChannelOverride, error) {
					return []models.ChannelOverride{{RoleID: testRoleID, Deny: int64(permissions.PermSendMessages)}, {RoleID: 10, Allow: int64(permissions.PermSendMessages)}}, nil
				}
				ch := channelMock()
				if kind != "guild" {
					ch = &mockChannelRepo{}
				}
				dm := &mockDMChannelRepo{GetByIDFn: func(context.Context, int64) (*models.DMChannel, error) {
					if kind == "unknown" {
						return nil, nil
					}
					return &models.DMChannel{Recipients: []models.User{{ID: 100}, {ID: 200}, {ID: 300}, {ID: 400}}}, nil
				}, IsRecipientFn: func(_ context.Context, c, u int64) (bool, error) { return u != 500, nil }}
				gw := &mockGateway{}
				svc := service.NewMessageService(&mockMessageRepo{}, ch, dm, testSnowflake(), gw, service.NewPermissionChecker(g, m, r, o))
				h := NewMessageHandler(svc)
				c, rec := newTestContext(http.MethodPost, "/", nil)
				c.SetParamNames("id")
				c.SetParamValues("2000")
				setAuthUser(c, actor)
				_ = h.Typing(c)
				want := 204
				if kind == "unknown" {
					want = 404
				} else if actor == 500 || (kind == "guild" && actor == 400) {
					want = 403
				}
				if rec.Code != want {
					t.Fatalf("status %d want %d: %s", rec.Code, want, rec.Body.String())
				}
				if want != 204 && len(gw.events) != 0 {
					t.Fatal("refused typing dispatched")
				}
				if want == 204 && kind == "dm" {
					if len(gw.events) != 3 {
						t.Fatal("wrong DM recipients")
					}
					for _, e := range gw.events {
						if e.UserID == actor || e.GuildID != 0 {
							t.Fatal("wrong DM fanout")
						}
					}
				}
			})
		}
	}
}

func TestPermissionRegressionMemberRoleReplacement(t *testing.T) {
	for _, position := range []int{1, 5, 6} {
		t.Run(fmt.Sprint(position), func(t *testing.T) {
			g, m, r, o := permMocks(permissions.PermManageRoles)
			r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
				return []models.Role{{Position: 5, Permissions: int64(permissions.PermManageRoles)}}, nil
			}
			r.GetByGuildIDFn = func(context.Context, int64) ([]models.Role, error) {
				return []models.Role{{ID: 10, Position: position}}, nil
			}
			changed := false
			m.AddRoleFn = func(context.Context, int64, int64, int64) error { changed = true; return nil }
			svc := service.NewMemberService(m, g, r, &mockGateway{}, service.NewPermissionChecker(g, m, r, o))
			ids := []int64{10}
			_, err := svc.UpdateMember(context.Background(), 1000, 3000, 4000, nil, &ids)
			if position < 5 {
				if err != nil || !changed {
					t.Fatalf("allowed: %v", err)
				}
			} else if err == nil || changed {
				t.Fatal("hierarchy changed membership")
			}
		})
	}
}

func TestPermissionRegressionTypingRoute(t *testing.T) {
	oldRegisterer := prometheus.DefaultRegisterer
	oldGatherer := prometheus.DefaultGatherer
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldRegisterer; prometheus.DefaultGatherer = oldGatherer })

	g, m, r, o := permMocks(permissions.PermSendMessages)
	m.GetByGuildAndUserFn = func(context.Context, int64, int64) (*models.Member, error) { return nil, nil }
	h := newMessageHandler(&mockMessageRepo{}, channelMock(), m, r, g, o, &mockGateway{})
	ts := auth.NewTokenService("local-routing-test")
	token, err := ts.GenerateAccessToken(500)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	SetupRouter(e, &Dependencies{Messages: h, TokenService: ts, Redis: newTestRedis(t)})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/2000/typing", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("typing route status %d, want 403", rec.Code)
	}
}
