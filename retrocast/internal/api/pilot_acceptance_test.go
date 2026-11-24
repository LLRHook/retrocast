package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPilotNonmemberReads(t *testing.T) {
	for _, op := range []string{"listChannels", "getChannel", "getMember"} {
		t.Run(op, func(t *testing.T) {
			m := &mockMemberRepo{GetByGuildAndUserFn: func(_ context.Context, g, u int64) (*models.Member, error) {
				if u == 200 {
					return &models.Member{GuildID: g, UserID: u}, nil
				}
				return nil, nil
			}}
			ch := &mockChannelRepo{GetByIDFn: func(context.Context, int64) (*models.Channel, error) { return &models.Channel{ID: 2, GuildID: 1}, nil }}
			c, rec := newTestContext(http.MethodGet, "/", nil)
			c.SetParamNames("id", "user_id")
			c.SetParamValues("1", "200")
			setAuthUser(c, 100)
			var err error
			if op == "getMember" {
				err = newMemberHandler(m, &mockGuildRepo{}, &mockRoleRepo{}, &mockGateway{}).GetMember(c)
			} else {
				h := newTestChannelHandler(ch, m, &mockGuildRepo{}, &mockRoleRepo{}, &mockChannelOverrideRepo{})
				if op == "listChannels" {
					err = h.ListChannels(c)
				} else {
					err = h.GetChannel(c)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if rec.Code != 404 {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			var detail ErrorResponse
			if err := decodeResponse(rec, &detail); err != nil {
				t.Fatal(err)
			}
			if detail.Error.Code != "NOT_FOUND" {
				t.Fatal(detail)
			}
		})
	}
}

func TestPilotInviteDefaults(t *testing.T) {
	g := &mockGuildRepo{GetByIDFn: func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1, OwnerID: 100}, nil }}
	h := newInviteHandler(&mockInviteRepo{}, g, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, &mockGateway{})
	c, rec := newTestContext(http.MethodPost, "/", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthUser(c, 100)
	before := time.Now()
	if err := h.CreateInvite(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 201 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var invite models.Invite
	if err := decodeResponse(rec, &invite); err != nil {
		t.Fatal(err)
	}
	if len(invite.Code) != 8 || invite.MaxUses != 0 || invite.ExpiresAt == nil || invite.ExpiresAt.Before(before.Add(24*time.Hour)) || invite.ExpiresAt.After(time.Now().Add(24*time.Hour)) {
		t.Fatalf("bad defaults: %+v", invite)
	}
}

func TestPilotInviteRejections(t *testing.T) {
	for _, mode := range []string{"expired", "banned"} {
		t.Run(mode, func(t *testing.T) {
			past := time.Now().Add(-time.Second)
			invites := &mockInviteRepo{GetByCodeFn: func(context.Context, string) (*models.Invite, error) {
				i := &models.Invite{GuildID: 1}
				if mode == "expired" {
					i.ExpiresAt = &past
				}
				return i, nil
			}}
			bans := &mockBanRepo{GetByGuildAndUserFn: func(context.Context, int64, int64) (*models.Ban, error) { return &models.Ban{}, nil }}
			h := newInviteHandler(invites, &mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, bans, &mockGateway{})
			c, rec := newTestContext(http.MethodPost, "/", nil)
			c.SetParamNames("code")
			c.SetParamValues("code")
			setAuthUser(c, 200)
			if err := h.AcceptInvite(c); err != nil {
				t.Fatal(err)
			}
			status, code := 410, "EXPIRED"
			if mode == "banned" {
				status, code = 403, "BANNED"
			}
			var detail ErrorResponse
			if err := decodeResponse(rec, &detail); err != nil {
				t.Fatal(err)
			}
			if rec.Code != status || detail.Error.Code != code {
				t.Fatalf("%d %+v", rec.Code, detail)
			}
		})
	}
}

func TestPilotMissingManagementPermissions(t *testing.T) {
	for _, op := range []string{"listInvites", "listBans", "unban", "deleteGuild"} {
		t.Run(op, func(t *testing.T) {
			g, m, r, _ := permMocks(0)
			c, rec := newTestContext(http.MethodDelete, "/", nil)
			c.SetParamNames("id", "user_id")
			c.SetParamValues("1000", "400")
			setAuthUser(c, testUserID)
			var err error
			switch op {
			case "listInvites":
				err = newInviteHandler(&mockInviteRepo{}, g, m, r, &mockBanRepo{}, &mockGateway{}).ListInvites(c)
			case "listBans":
				err = newBanHandler(g, m, r, &mockBanRepo{}, &mockGateway{}).ListBans(c)
			case "unban":
				err = newBanHandler(g, m, r, &mockBanRepo{}, &mockGateway{}).UnbanMember(c)
			case "deleteGuild":
				r.GetByMemberFn = func(context.Context, int64, int64) ([]models.Role, error) {
					return []models.Role{{Permissions: 1 << 31}}, nil
				}
				err = newTestGuildHandler(g, &mockChannelRepo{}, m, r).DeleteGuild(c)
			}
			if err != nil {
				t.Fatal(err)
			}
			var detail ErrorResponse
			if err := decodeResponse(rec, &detail); err != nil {
				t.Fatal(err)
			}
			code := "MISSING_PERMISSIONS"
			if op == "deleteGuild" {
				code = "FORBIDDEN"
			}
			if rec.Code != 403 || detail.Error.Code != code {
				t.Fatalf("%d %+v", rec.Code, detail)
			}
		})
	}
}

func TestPilotListTwoGuilds(t *testing.T) {
	g := &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}, {ID: 2}}, nil }}
	c, rec := newTestContext(http.MethodGet, "/", nil)
	setAuthUser(c, 100)
	if err := newTestGuildHandler(g, &mockChannelRepo{}, &mockMemberRepo{}, &mockRoleRepo{}).ListMyGuilds(c); err != nil {
		t.Fatal(err)
	}
	var guilds []models.Guild
	if err := decodeResponse(rec, &guilds); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || len(guilds) != 2 || guilds[0].ID != 1 || guilds[1].ID != 2 {
		t.Fatalf("%d %+v", rec.Code, guilds)
	}
}

func TestPilotLogoutRevocation(t *testing.T) {
	rc := newTestRedis(t)
	if err := rc.StoreRefreshToken(context.Background(), "pilot-refresh", 100, time.Hour); err != nil {
		t.Fatal(err)
	}
	h := NewAuthHandler(service.NewAuthService(nil, auth.NewTokenService("pilot-test"), rc, nil))
	invoke := func(fn echo.HandlerFunc) int {
		c, rec := newTestContext(http.MethodPost, "/", strings.NewReader(`{"refresh_token":"pilot-refresh"}`))
		if err := fn(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code == 204 {
			if rec.Body.Len() != 0 {
				t.Fatal("204 body")
			}
		} else {
			var detail ErrorResponse
			if err := decodeResponse(rec, &detail); err != nil {
				t.Fatal(err)
			}
			if detail.Error.Code != "INVALID_TOKEN" {
				t.Fatal(detail)
			}
		}
		return rec.Code
	}
	if status := invoke(h.Logout); status != 204 {
		t.Fatal(status)
	}
	if status := invoke(h.Refresh); status != 401 {
		t.Fatal(status)
	}
}

func TestPilotRedisDownRouter(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL required")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	mr := miniredis.RunT(t)
	rc, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	mr.Close()
	g, m, r, o := permMocks(permissions.PermViewChannel | permissions.PermReadMessageHistory)
	h := newMessageHandler(&mockMessageRepo{}, channelMock(), m, r, g, o, &mockGateway{})
	old := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	defer func() { prometheus.DefaultRegisterer = old }()
	tokens := auth.NewTokenService("pilot-redis-down")
	token, err := tokens.GenerateAccessToken(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	SetupRouter(e, &Dependencies{Messages: h, TokenService: tokens, Redis: rc, Pool: pool})
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/health", 503}, {"POST", "/api/v1/auth/login", 503}, {"GET", "/api/v1/invites/pilot", 503}, {"GET", "/api/v1/channels/2000/messages", 200}} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.status == 200 {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("%d %s", rec.Code, rec.Body.String())
			}
			if tc.path == "/health" {
				var body map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["status"] != "error" || body["component"] != "redis" {
					t.Fatal(body)
				}
			} else if tc.status == 503 {
				var body ErrorResponse
				if err := decodeResponse(rec, &body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != "SERVICE_UNAVAILABLE" {
					t.Fatal(body)
				}
			}
		})
	}
}

func TestPilotModerationBoundaries(t *testing.T) {
	for _, op := range []string{"kick", "ban"} {
		for _, mode := range []string{"equal", "owner", "self", "everyone"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				g, m, r, o := permMocks(permissions.PermKickMembers | permissions.PermBanMembers)
				g.GetByIDFn = func(context.Context, int64) (*models.Guild, error) { return &models.Guild{ID: 1000, OwnerID: 100}, nil }
				m.GetByGuildAndUserFn = func(_ context.Context, g, u int64) (*models.Member, error) {
					return &models.Member{GuildID: g, UserID: u}, nil
				}
				r.GetByMemberFn = func(_ context.Context, g, u int64) ([]models.Role, error) {
					if u == testUserID {
						position := 1
						if mode == "equal" {
							position = 2
						}
						return []models.Role{{Position: position, Permissions: int64(permissions.PermKickMembers | permissions.PermBanMembers)}}, nil
					}
					if mode == "equal" {
						return []models.Role{{Position: 2}}, nil
					}
					return nil, nil
				}
				target := "400"
				if mode == "owner" {
					target = "100"
				}
				if mode == "self" {
					target = fmt.Sprint(testUserID)
				}
				c, rec := newTestContext(http.MethodDelete, "/", nil)
				c.SetParamNames("id", "user_id")
				c.SetParamValues("1000", target)
				setAuthUser(c, testUserID)
				var err error
				if op == "kick" {
					err = NewMemberHandler(service.NewMemberService(m, g, r, &mockGateway{}, service.NewPermissionChecker(g, m, r, o))).KickMember(c)
				} else {
					err = newBanHandler(g, m, r, &mockBanRepo{}, &mockGateway{}).BanMember(c)
				}
				if err != nil {
					t.Fatal(err)
				}
				status, code := 204, ""
				switch mode {
				case "equal":
					status, code = 403, "ROLE_HIERARCHY"
				case "owner":
					status, code = 403, "FORBIDDEN"
				case "self":
					status, code = 400, "CANNOT_KICK_SELF"
					if op == "ban" {
						code = "CANNOT_BAN_SELF"
					}
				}
				if rec.Code != status {
					t.Fatalf("status %d want %d: %s", rec.Code, status, rec.Body.String())
				}
				if code != "" {
					var detail ErrorResponse
					if err := decodeResponse(rec, &detail); err != nil {
						t.Fatal(err)
					}
					if detail.Error.Code != code {
						t.Fatal(detail)
					}
				}
			})
		}
	}
}
