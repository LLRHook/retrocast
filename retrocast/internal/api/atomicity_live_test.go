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
	"github.com/victorivanov/retrocast/internal/models"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/service"
)

// A database per run isolates bootstrap and preserves migrations pinned to public.
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
	databaseName := fmt.Sprintf("l23_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+databaseName); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP DATABASE "+databaseName) }()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = databaseName
	cfg.ConnConfig.RuntimeParams["deadlock_timeout"] = "100ms"
	cfg.MaxConns = 32
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
	t.Run("bootstrap lock", func(t *testing.T) {
		type result struct {
			first bool
			err   error
		}
		done := make(chan result, 1)
		var early *result
		err := database.InTransaction(ctx, users, func(txctx context.Context) error {
			first, err := database.LockRegistration(txctx, users)
			if err != nil {
				return err
			}
			if !first {
				return fmt.Errorf("expected empty instance")
			}
			if err := users.Create(txctx, &models.User{ID: 99, Username: "lockfixture", DisplayName: "fixture", PasswordHash: "fixture", CreatedAt: time.Now()}); err != nil {
				return err
			}
			started := make(chan struct{})
			go func() {
				var first bool
				err := database.InTransaction(ctx, users, func(other context.Context) error {
					close(started)
					var err error
					first, err = database.LockRegistration(other, users)
					return err
				})
				done <- result{first, err}
			}()
			<-started
			select {
			case r := <-done:
				early = &r
			case <-time.After(250 * time.Millisecond):
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		var r result
		if early != nil {
			r = *early
		} else {
			select {
			case r = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("second bootstrap transaction did not finish")
			}
		}
		exec("DELETE FROM users WHERE id=99")
		if early != nil {
			t.Fatal("second bootstrap lock returned before first transaction committed")
		}
		if r.err != nil || r.first {
			t.Fatalf("second transaction: first=%v err=%v", r.first, r.err)
		}
	})
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
		events, subs := len(gw.events), len(gw.subscriptions)
		check(t, register("failedjoin", "failuse"), 500, "INTERNAL")
		if len(gw.events) != events || len(gw.subscriptions) != subs {
			t.Fatal("failed registration published gateway state")
		}
		exec("DROP TRIGGER fail_use ON invites; DROP FUNCTION fail_use()")
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE username='failedjoin'").Scan(&n)
		if n != 0 {
			t.Fatal("failed join persisted account")
		}
	})
	// A deferred failure exercises the boundary between successful business writes
	// and COMMIT: neither acceptance path may publish before that boundary.
	for _, registration := range []bool{false, true} {
		t.Run(fmt.Sprintf("commit rollback registration=%v", registration), func(t *testing.T) {
			code := fmt.Sprintf("cf%v", registration)
			makeInvite(code, 5, time.Now().Add(time.Hour))
			exec("INSERT INTO users(id,username,display_name,password_hash) VALUES(98,'rollbackjoiner','fixture','fixture') ON CONFLICT DO NOTHING")
			exec(`CREATE FUNCTION fail_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected commit failure'; END $$; CREATE CONSTRAINT TRIGGER fail_commit AFTER UPDATE ON invites DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION fail_commit()`)
			events, subs := len(gw.events), len(gw.subscriptions)
			if registration {
				check(t, register("commitfailure", code), 500, "INTERNAL")
			} else {
				c, r := newTestContext(http.MethodPost, "/api/v1/invites/"+code, nil)
				c.SetParamNames("code")
				c.SetParamValues(code)
				setAuthUser(c, 98)
				if err := NewInviteHandler(invites).AcceptInvite(c); err != nil {
					t.Fatal(err)
				}
				check(t, r, 500, "INTERNAL")
			}
			exec("DROP TRIGGER fail_commit ON invites; DROP FUNCTION fail_commit()")
			if len(gw.events) != events || len(gw.subscriptions) != subs {
				t.Fatal("published gateway state before commit")
			}
			var n, uses int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM members WHERE guild_id=$1 AND user_id=98", guild.ID).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Fatal("membership escaped rollback")
			}
			if err := pool.QueryRow(ctx, "SELECT uses FROM invites WHERE code=$1", code).Scan(&uses); err != nil {
				t.Fatal(err)
			}
			if uses != 0 {
				t.Fatal("invite use escaped rollback")
			}
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE username='commitfailure'").Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Fatal("account escaped rollback")
			}
		})
	}
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
		const racers = 256
		for round := 0; round < 5; round++ {
			token, err := auth.NewTokenService("local-atomicity-fixture").GenerateRefreshToken()
			if err != nil {
				t.Fatal(err)
			}
			if err := rdb.StoreRefreshToken(ctx, token, ownerID, time.Minute); err != nil {
				t.Fatal(err)
			}
			counts := concurrent(racers, func(i int) int {
				c, r := newTestContext(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(fmt.Sprintf(`{"refresh_token":%q}`, token)))
				if err := h.Refresh(c); err != nil {
					t.Error(err)
				}
				return r.Code
			})
			if counts[200] != 1 || counts[401] != racers-1 {
				t.Fatalf("refresh round %d: %v", round, counts)
			}
		}
	})

	t.Run("guild delete race", func(t *testing.T) {
		// Pause DELETE after it locks the guild, while joiners reach the same row.
		// This forces the old invite-first implementation into its lock inversion.
		const barrier int64 = 72463002
		exec(`CREATE FUNCTION pause_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(72463002); RETURN OLD; END $$; CREATE TRIGGER pause_delete BEFORE DELETE ON guilds FOR EACH ROW EXECUTE FUNCTION pause_delete()`)
		defer exec("DROP TRIGGER pause_delete ON guilds; DROP FUNCTION pause_delete()")
		waitBlocked := func(pid uint32) {
			t.Helper()
			deadline := time.Now().Add(5 * time.Second)
			for {
				var n int
				if err := pool.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid))", pid).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n > 0 {
					return
				}
				if time.Now().After(deadline) {
					t.Fatal("no transaction waited on expected lock")
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
		gh := NewGuildHandler(gs)
		ih := NewInviteHandler(invites)
		for round := 0; round < 10; round++ {
			g, err := gs.CreateGuild(ctx, ownerID, fmt.Sprintf("Delete race %d", round))
			if err != nil {
				t.Fatal(err)
			}
			code := fmt.Sprintf("delete%d", round)
			exec("INSERT INTO invites(code,guild_id,creator_id) VALUES($1,$2,$3)", code, g.ID, ownerID)
			guard, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := guard.Exec(ctx, "SELECT pg_advisory_lock($1)", barrier); err != nil {
				guard.Release()
				t.Fatal(err)
			}
			guardPID := guard.Conn().PgConn().PID()
			deleted := make(chan int, 1)
			go func() {
				c, r := newTestContext(http.MethodDelete, "/api/v1/guilds/"+fmt.Sprint(g.ID), nil)
				c.SetParamNames("id")
				c.SetParamValues(fmt.Sprint(g.ID))
				setAuthUser(c, ownerID)
				if err := gh.DeleteGuild(c); err != nil {
					t.Error(err)
				}
				deleted <- r.Code
			}()
			// Always release the session lock, including assertion failures.
			func() {
				defer guard.Release()
				locked := true
				defer func() {
					if locked {
						_, _ = guard.Exec(ctx, "SELECT pg_advisory_unlock($1)", barrier)
					}
				}()
				waitBlocked(guardPID)
				var deletePID uint32
				if err := pool.QueryRow(ctx, "SELECT pid FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid)) LIMIT 1", guardPID).Scan(&deletePID); err != nil {
					t.Fatal(err)
				}
				joined := make(chan int, 12)
				names := make([]string, 4)
				for i := 0; i < 8; i++ {
					id := int64(1000 + round*8 + i)
					exec("INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'fixture','fixture')", id, fmt.Sprintf("deletejoin%d_%d", round, i))
					go func() {
						c, r := newTestContext(http.MethodPost, "/api/v1/invites/"+code, nil)
						c.SetParamNames("code")
						c.SetParamValues(code)
						setAuthUser(c, id)
						if err := ih.AcceptInvite(c); err != nil {
							t.Error(err)
						}
						joined <- r.Code
					}()
				}
				for i := 0; i < 4; i++ {
					names[i] = fmt.Sprintf("deletereg%d_%d", round, i)
					name := names[i]
					go func() { joined <- register(name, code).Code }()
				}
				waitBlocked(deletePID)
				if _, err := guard.Exec(ctx, "SELECT pg_advisory_unlock($1)", barrier); err != nil {
					t.Fatal(err)
				}
				locked = false
				if status := <-deleted; status != 204 {
					t.Errorf("round %d delete status %d", round, status)
				}
				for i := 0; i < 12; i++ {
					status := <-joined
					if status != 200 && status != 403 && status != 404 {
						t.Errorf("round %d join status %d", round, status)
					}
				}
				var n int
				if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM guilds WHERE id=$1)+(SELECT count(*) FROM members WHERE guild_id=$1)+(SELECT count(*) FROM invites WHERE guild_id=$1)+(SELECT count(*) FROM roles WHERE guild_id=$1)+(SELECT count(*) FROM channels WHERE guild_id=$1)`, g.ID).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n != 0 {
					t.Errorf("round %d retained %d guild rows", round, n)
				}
				if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE username=ANY($1::text[])", names).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n != 0 {
					t.Errorf("round %d failed registrations left accounts", round)
				}
			}()
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
