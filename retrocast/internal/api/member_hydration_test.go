package api

import (
	"context"
	"errors"
	"github.com/victorivanov/retrocast/internal/models"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/service"
)

type l40R2CountTracer struct {
	mu   sync.Mutex
	sqls []string
}

func (c *l40R2CountTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	c.mu.Lock()
	c.sqls = append(c.sqls, strings.Join(strings.Fields(d.SQL), " "))
	c.mu.Unlock()
	return ctx
}
func (c *l40R2CountTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func (c *l40R2CountTracer) take() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.sqls
	c.sqls = nil
	return out
}

var l40R2Tracer = &l40R2CountTracer{}

func TestL40R2QueryCount(t *testing.T) {
	e, ownerToken, pool := pilotContractServerWithTracer(t, l40R2Tracer)
	ctx := context.Background()
	perms := permissions.PermViewChannel | permissions.PermSendMessages | permissions.PermReadMessageHistory
	if _, err := pool.Exec(ctx, "UPDATE roles SET permissions=$1 WHERE id=30", int64(perms)); err != nil {
		t.Fatal(err)
	}
	memberToken, err := auth.NewTokenService("pilot-contract-fixture").GenerateAccessToken(200)
	if err != nil {
		t.Fatal(err)
	}
	do := func(label, method, path, body, token string) {
		l40R2Tracer.take()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		sqls := l40R2Tracer.take()
		t.Logf("L40R2COUNT %s status=%d queries=%d", label, rec.Code, len(sqls))
		expected := map[string]int{"send-member": 12, "history-member": 9, "typing-member": 7, "send-owner": 7, "history-owner": 4, "getMember-member": 5, "listMembers-member": 6, "listChannels-member": 3, "invitePreview-anon": 5}[label]
		if rec.Code < 200 || rec.Code >= 300 || len(sqls) != expected {
			t.Fatalf("%s status=%d queries=%d want=%d", label, rec.Code, len(sqls), expected)
		}
		identities := 0
		for _, sql := range sqls {
			if strings.Contains(sql, "FROM users WHERE id = ANY") {
				identities++
			}
		}
		wantIdentities := 0
		if label == "getMember-member" || label == "listMembers-member" {
			wantIdentities = 1
		}
		if identities != wantIdentities {
			t.Fatalf("%s identity queries=%d want=%d", label, identities, wantIdentities)
		}
		for i, s := range sqls {
			if len(s) > 140 {
				s = s[:140]
			}
			t.Logf("L40R2SQL %s %02d %s", label, i+1, s)
		}
	}
	do("send-member", "POST", "/api/v1/channels/10/messages", `{"content":"x"}`, memberToken)
	do("history-member", "GET", "/api/v1/channels/10/messages", "", memberToken)
	do("typing-member", "POST", "/api/v1/channels/10/typing", "", memberToken)
	do("send-owner", "POST", "/api/v1/channels/10/messages", `{"content":"y"}`, ownerToken)
	do("history-owner", "GET", "/api/v1/channels/10/messages", "", ownerToken)
	do("getMember-member", "GET", "/api/v1/guilds/1/members/100", "", memberToken)
	do("listMembers-member", "GET", "/api/v1/guilds/1/members", "", memberToken)
	do("listChannels-member", "GET", "/api/v1/guilds/1/channels", "", memberToken)
	do("invitePreview-anon", "GET", "/api/v1/invites/pilot", "", "")
	checker := service.NewPermissionChecker(database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool), database.NewChannelOverrideRepository(pool))
	l40R2Tracer.take()
	if err := checker.RequireChannelPermission(ctx, 1, 10, 200, permissions.PermViewChannel); err != nil {
		t.Fatal(err)
	}
	sqls := l40R2Tracer.take()
	t.Logf("L40R2COUNT gateway-visibility status=allowed queries=%d", len(sqls))
	if len(sqls) != 6 {
		t.Fatalf("gateway visibility queries=%d want base 6", len(sqls))
	}
	for _, sql := range sqls {
		if strings.Contains(sql, "FROM users WHERE id = ANY") {
			t.Fatal("gateway visibility hydrated identity")
		}
	}

}

type failingMemberHydrator struct{ database.MemberRepository }

func (r failingMemberHydrator) HydrateUsers(ctx context.Context, members []*models.Member) error {
	if err := r.MemberRepository.HydrateUsers(ctx, members); err != nil {
		return err
	}
	return errors.New("injected identity failure")
}
func TestMemberEdgeHydrationTransaction(t *testing.T) {
	_, _, pool := pilotContractServer(t)
	ctx := context.Background()
	users, guilds, members, roles := database.NewUserRepository(pool), database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool)
	gw := &mockGateway{}
	perms := service.NewPermissionChecker(guilds, members, roles, database.NewChannelOverrideRepository(pool))
	repo := database.NewInviteRepository(pool)
	bans := database.NewBanRepository(pool)
	invites := service.NewInviteService(repo, guilds, members, bans, gw, perms)
	as := service.NewAuthService(users, auth.NewTokenService("edge-transaction"), newTestRedis(t), testSnowflake())
	as.ConfigureRegistration("invite", invites)
	c, rec := newTestContext("POST", "/", strings.NewReader(`{"username":"r2new","password":"password123","invite_code":"pilot"}`))
	if err := NewAuthHandler(as).Register(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 {
		t.Fatalf("registration %d %s", rec.Code, rec.Body.String())
	}
	member := assertGatewayPayload(t, gw, "GUILD_MEMBER_ADD", map[string]any{"guild_id": "1"}).(*models.Member)
	if member.User == nil || member.User.Username != "r2new" {
		t.Fatal("registration event lost uncommitted identity")
	}
	var before int
	if err := pool.QueryRow(ctx, "SELECT uses FROM invites WHERE code='pilot'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	gw.events = nil
	broken := service.NewInviteService(repo, guilds, failingMemberHydrator{members}, bans, gw, perms)
	if _, err := broken.AcceptInvite(ctx, "pilot", 300); err == nil {
		t.Fatal("hydration failure accepted")
	}
	var rows, after int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM members WHERE guild_id=1 AND user_id=300").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT uses FROM invites WHERE code='pilot'").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || after != before || len(gw.events) != 0 {
		t.Fatalf("join not rolled back: rows %d uses %d -> %d events %d", rows, before, after, len(gw.events))
	}
}
