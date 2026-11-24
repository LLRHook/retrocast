package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
	"os"
	"path/filepath"
	"strings"
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
	exec(`CREATE FUNCTION reject_ban_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'l28 commit failure'; END $$;
 CREATE CONSTRAINT TRIGGER fail_insert AFTER INSERT ON bans DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_ban_insert()`)
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
	exec("DROP TRIGGER fail_insert ON bans")
	if err := svc.BanMember(ctx, 1, 100, 200, &reason); err != nil {
		t.Fatal(err)
	}
	member, _ = members.GetByGuildAndUser(ctx, 1, 200)
	ban, _ = bans.GetByGuildAndUser(ctx, 1, 200)
	if member != nil || ban == nil || ban.Reason == nil || *ban.Reason != reason {
		t.Fatal("ban not committed")
	}

	inviteRepo := database.NewInviteRepository(pool)
	if err := inviteRepo.Create(ctx, &models.Invite{Code: "l40join1", GuildID: 1, CreatorID: 100, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	inviteSvc := service.NewInviteService(inviteRepo, guilds, members, bans, gw, perms)
	if _, err := inviteSvc.AcceptInvite(ctx, "l40join1", 200); err == nil {
		t.Fatal("banned member rejoined")
	} else {
		c, rec := newTestContext("POST", "/", nil)
		_ = mapServiceError(c, err)
		var detail ErrorResponse
		if err := decodeResponse(rec, &detail); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 403 || detail.Error.Code != "BANNED" {
			t.Fatal(rec.Code, detail)
		}
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
	if _, err := inviteSvc.AcceptInvite(ctx, "l40join1", 200); err != nil {
		t.Fatal("unbanned member cannot rejoin", err)
	}
	added := assertGatewayPayload(t, gw, "GUILD_MEMBER_ADD", map[string]any{"guild_id": "1", "user_id": "200"}).(*models.Member)
	if added.User == nil || added.User.Username != "target" {
		t.Fatal("add event identity absent")
	}

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

	page, err := members.GetByGuildID(ctx, 1, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range page {
		if m.UserID == 200 {
			t.Fatal("kicked member remains in list")
		}
	}
	if _, err := inviteSvc.AcceptInvite(ctx, "l40join1", 200); err != nil {
		t.Fatal("kicked member cannot rejoin", err)
	}
	t.Run("guild parent lock precedes child mutations", func(t *testing.T) {
		exec("INSERT INTO guilds(id,name,owner_id) VALUES(2,'Locked',100)")
		exec("INSERT INTO members(guild_id,user_id) VALUES(2,100),(2,200)")
		held, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Rollback(ctx) }()
		if _, err = held.Exec(ctx, "SELECT id FROM guilds WHERE id=2 FOR UPDATE"); err != nil {
			t.Fatal(err)
		}
		opctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- svc.BanMember(opctx, 2, 100, 200, &reason) }()
		// Inspect the blocked statement: an unlocked mutant can also block later on
		// the ban's parent FK, after deleting the member. That must not pass.
		deadline := time.After(2 * time.Second)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		waiting := false
		for !waiting {
			select {
			case err := <-done:
				t.Fatalf("ban bypassed held parent lock: %v", err)
			case <-deadline:
				t.Fatal("ban did not wait on the guild FOR UPDATE lock")
			case <-ticker.C:
				err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query=$1)`, "SELECT id FROM guilds WHERE id=$1 FOR UPDATE").Scan(&waiting)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		if err = held.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if err = <-done; err != nil {
			t.Fatal(err)
		}
	})
	t.Run("deleted guild returns unknown guild", func(t *testing.T) {
		exec("INSERT INTO guilds(id,name,owner_id) VALUES(3,'Deleted',100)")
		exec("INSERT INTO members(guild_id,user_id) VALUES(3,100),(3,200)")
		raceRoles := &mockRoleRepo{GetByMemberFn: func(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
			rs, err := roles.GetByMember(ctx, guildID, userID)
			if err == nil && userID == 200 {
				err = guilds.Delete(ctx, guildID)
			}
			return rs, err
		}}
		raceSvc := service.NewBanService(guilds, members, raceRoles, bans, gw, perms)
		before := len(gw.events)
		err := raceSvc.BanMember(ctx, 3, 100, 200, &reason)
		var se *service.ServiceError
		if !errors.As(err, &se) || !errors.Is(err, service.ErrNotFound) || se.Code != "UNKNOWN_GUILD" {
			t.Fatalf("want 404 UNKNOWN_GUILD, got %v", err)
		}
		if len(gw.events) != before {
			t.Fatal("deleted guild published events")
		}
		c, rec := newTestContext("PUT", "/", strings.NewReader("{}"))
		if err := mapServiceError(c, err); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 404 {
			t.Fatal("wrong HTTP status")
		}
	})

}
