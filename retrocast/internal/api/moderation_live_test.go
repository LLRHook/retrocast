package api

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/service"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestModerationAtomicityLive(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("REDIS_URL") == "" {
		t.Skip("DATABASE_URL and REDIS_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	databaseName := fmt.Sprintf("l28_%d", time.Now().UnixNano())
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

	exec("INSERT INTO users(id,username,display_name,password_hash) VALUES(100,'owner','Owner','hash'),(200,'target','Target','hash')")
	exec("INSERT INTO guilds(id,name,owner_id) VALUES(1,'Moderation',100)")
	exec("INSERT INTO members(guild_id,user_id) VALUES(1,100),(1,200)")
	guilds, members, roles, bans := database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool), database.NewBanRepository(pool)
	gw := &mockGateway{}
	perms := service.NewPermissionChecker(guilds, members, roles, database.NewChannelOverrideRepository(pool))
	svc := service.NewBanService(guilds, members, roles, bans, gw, perms)
	exec(`CREATE FUNCTION reject_member_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'l28 commit failure'; END $$;
 CREATE CONSTRAINT TRIGGER fail_delete AFTER DELETE ON members DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_member_delete()`)
	reason := "spam"
	if err := svc.BanMember(ctx, 1, 100, 200, &reason); err == nil {
		t.Fatal("commit failure accepted")
	}
	member, err := members.GetByGuildAndUser(ctx, 1, 200)
	if err != nil || member == nil {
		t.Fatal("member not restored")
	}
	ban, err := bans.GetByGuildAndUser(ctx, 1, 200)
	if err != nil || ban != nil {
		t.Fatal("ban half applied")
	}
	if len(gw.events) != 0 || len(gw.unsubscriptions) != 0 {
		t.Fatal("published before commit")
	}
	exec("DROP TRIGGER fail_delete ON members")
	if err := svc.BanMember(ctx, 1, 100, 200, &reason); err != nil {
		t.Fatal(err)
	}
	member, _ = members.GetByGuildAndUser(ctx, 1, 200)
	ban, _ = bans.GetByGuildAndUser(ctx, 1, 200)
	if member != nil || ban == nil || ban.Reason == nil || *ban.Reason != reason {
		t.Fatal("ban not committed")
	}
	found := false
	for _, e := range gw.events {
		if e.UserID == 200 && e.Event == gateway.EventGuildDelete {
			found = true
		}
	}
	if !found {
		t.Fatal("removed user's clients not notified")
	}
	if err := svc.UnbanMember(ctx, 1, 100, 200); err != nil {
		t.Fatal(err)
	}
	if err := svc.UnbanMember(ctx, 1, 100, 200); err != nil {
		t.Fatal("unban not idempotent")
	}
	exec("INSERT INTO members(guild_id,user_id) VALUES(1,200)")
	gw.events = nil
	ms := service.NewMemberService(members, guilds, roles, gw, perms)
	if err := ms.KickMember(ctx, 1, 100, 200); err != nil {
		t.Fatal(err)
	}
	found = false
	for _, e := range gw.events {
		if e.UserID == 200 && e.Event == gateway.EventGuildDelete {
			found = true
		}
	}
	if !found {
		t.Fatal("kicked user's clients not notified")
	}
}
