package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/service"
)

// All state is isolated from other packages, including the empty-instance test.
func TestRegistrationAtomicityLive(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("REDIS_URL") == "" {
		t.Skip("DATABASE_URL and REDIS_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("l23_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	files, _ := filepath.Glob("../../migrations/*.up.sql")
	for _, f := range files {
		b, e := os.ReadFile(f)
		if e != nil {
			t.Fatal(e)
		}
		exec(string(b))
	}
	rdb, err := redisclient.NewClient(os.Getenv("REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rdb.Close() }()
	users, guilds, members, roles := database.NewUserRepository(pool), database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool)
	sf := testSnowflake()
	gw := &mockGateway{}
	perms := service.NewPermissionChecker(guilds, members, roles, database.NewChannelOverrideRepository(pool))
	invites := service.NewInviteService(database.NewInviteRepository(pool), guilds, members, database.NewBanRepository(pool), gw, perms)
	as := service.NewAuthService(users, auth.NewTokenService("local-atomicity-fixture"), rdb, sf)
	as.ConfigureRegistration("invite", invites)
	h := NewAuthHandler(as)
	register := func(name, code string) *httptest.ResponseRecorder {
		c, r := newTestContext(http.MethodPost, "/api/v1/auth/register", strings.NewReader(fmt.Sprintf(`{"username":%q,"password":"password123","invite_code":%q}`, name, code)))
		if e := h.Register(c); e != nil {
			t.Error(e)
		}
		return r
	}
	check := func(t *testing.T, r *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		if r.Code != status || (code != "" && !strings.Contains(r.Body.String(), `"code":"`+code+`"`)) {
			t.Fatalf("status=%d, expected %d code %s", r.Code, status, code)
		}
	}
	concurrent := func(n int, fn func(int) int) map[int]int {
		var wg sync.WaitGroup
		start := make(chan struct{})
		out := make(chan int, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) { defer wg.Done(); <-start; out <- fn(i) }(i)
		}
		close(start)
		wg.Wait()
		close(out)
		counts := map[int]int{}
		for x := range out {
			counts[x]++
		}
		return counts
	}
	t.Run("bootstrap", func(t *testing.T) {
		counts := concurrent(2, func(i int) int { return register(fmt.Sprintf("first%d", i), "").Code })
		if counts[200] != 1 || counts[403] != 1 {
			t.Fatalf("bootstrap %v", counts)
		}
		check(t, register("missing", ""), 403, "INVITE_REQUIRED")
	})
	as.ConfigureRegistration("open", invites)
	owner := register("owner", "")
	check(t, owner, 200, "")
	var response struct {
		Data struct {
			User struct {
				ID string `json:"id"`
			}
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err = json.Unmarshal(owner.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var ownerID int64
	if _, err = fmt.Sscan(response.Data.User.ID, &ownerID); err != nil {
		t.Fatal(err)
	}
	gs := service.NewGuildService(guilds, database.NewChannelRepository(pool), members, roles, sf, gw, perms)
	guild, err := gs.CreateGuild(ctx, ownerID, "Atomic guild")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("guild rollback", func(t *testing.T) {
		exec(`CREATE FUNCTION fail_channel() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$; CREATE TRIGGER fail_channel BEFORE INSERT ON channels FOR EACH ROW EXECUTE FUNCTION fail_channel()`)
		events, subs := len(gw.events), len(gw.subscriptions)
		if _, e := gs.CreateGuild(ctx, ownerID, "Failed guild"); e == nil {
			t.Fatal("expected failure")
		}
		var n int
		if e := pool.QueryRow(ctx, "SELECT count(*) FROM guilds WHERE name='Failed guild'").Scan(&n); e != nil || n != 0 {
			t.Fatalf("partial guild %d %v", n, e)
		}
		if len(gw.events) != events || len(gw.subscriptions) != subs {
			t.Fatal("published before commit")
		}
		exec("DROP TRIGGER fail_channel ON channels; DROP FUNCTION fail_channel()")
	})
	makeInvite := func(code string, max int, expires time.Time) {
		exec("INSERT INTO invites(code,guild_id,creator_id,max_uses,expires_at) VALUES($1,$2,$3,$4,$5)", code, guild.ID, ownerID, max, expires)
	}
	as.ConfigureRegistration("invite", invites)
	t.Run("registration invite and rollback", func(t *testing.T) {
		makeInvite("register", 1, time.Now().Add(time.Hour))
		check(t, register("invited", "register"), 200, "")
		check(t, register("exhausted", "register"), 403, "INVALID_INVITE")
		check(t, register("unknown", "absent"), 403, "INVALID_INVITE")
		makeInvite("expired", 1, time.Now().Add(-time.Hour))
		check(t, register("expired", "expired"), 403, "INVALID_INVITE")
		var n int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE username IN ('exhausted','unknown','expired')").Scan(&n)
		if n != 0 {
			t.Fatal("failed registration persisted")
		}
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM members m JOIN users u ON u.id=m.user_id WHERE u.username='invited' AND guild_id=$1", guild.ID).Scan(&n)
		if n != 1 {
			t.Fatal("membership missing")
		}
		// Failure after membership insert must roll back account, membership and use.
		makeInvite("failuse", 1, time.Now().Add(time.Hour))
		exec(`CREATE FUNCTION fail_use() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$; CREATE TRIGGER fail_use BEFORE UPDATE ON invites FOR EACH ROW EXECUTE FUNCTION fail_use()`)
		check(t, register("failedjoin", "failuse"), 500, "INTERNAL")
		exec("DROP TRIGGER fail_use ON invites; DROP FUNCTION fail_use()")
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE username='failedjoin'").Scan(&n)
		if n != 0 {
			t.Fatal("failed join persisted account")
		}
	})
	t.Run("invite race", func(t *testing.T) {
		makeInvite("race", 3, time.Now().Add(time.Hour))
		for i := 0; i < 24; i++ {
			exec("INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'race','fixture')", int64(100+i), fmt.Sprintf("race%d", i))
		}
		counts := concurrent(24, func(i int) int {
			_, e := invites.AcceptInvite(ctx, "race", int64(100+i))
			if e == nil {
				return 200
			}
			if se, ok := e.(*service.ServiceError); ok && se.Code == "MAX_USES" {
				return 410
			}
			return 500
		})
		if counts[200] != 3 || counts[410] != 21 {
			t.Fatalf("invite race %v", counts)
		}
		var uses, n int
		_ = pool.QueryRow(ctx, "SELECT uses FROM invites WHERE code='race'").Scan(&uses)
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM members WHERE user_id BETWEEN 100 AND 123").Scan(&n)
		if uses != 3 || n != 3 {
			t.Fatalf("uses=%d members=%d", uses, n)
		}
	})
	t.Run("refresh race", func(t *testing.T) {
		counts := concurrent(32, func(i int) int {
			c, r := newTestContext(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(fmt.Sprintf(`{"refresh_token":%q}`, response.Data.RefreshToken)))
			if e := h.Refresh(c); e != nil {
				t.Error(e)
			}
			return r.Code
		})
		if counts[200] != 1 || counts[401] != 31 {
			t.Fatalf("refresh race %v", counts)
		}
	})
	t.Run("closed preserves login", func(t *testing.T) {
		as.ConfigureRegistration("closed", invites)
		check(t, register("closed", ""), 403, "REGISTRATION_CLOSED")
		c, r := newTestContext(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"owner","password":"password123"}`))
		if e := h.Login(c); e != nil {
			t.Fatal(e)
		}
		check(t, r, 200, "")
	})
}
