package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/snowflake"
	"os"
	"testing"
	"time"
)

// The assignment holds the FK's member KEY SHARE lock until we observe the
// moderation connection waiting on it. No scheduling delay decides the outcome.
func TestModerationConcurrentRoleAssignment(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	for _, op := range []string{"ban", "kick"} {
		t.Run(op, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			sf, _ := snowflake.NewGenerator(18, 19)
			owner, actor, target, guild, low, high := sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64()
			exec := func(q string, args ...any) {
				t.Helper()
				if _, err := pool.Exec(ctx, q, args...); err != nil {
					t.Fatal(err)
				}
			}
			for _, id := range []int64{owner, actor, target} {
				exec("INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'race','fixture')", id, sf.Generate().String())
			}
			defer func() {
				_, _ = pool.Exec(context.Background(), "DELETE FROM guilds WHERE id=$1", guild)
				_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id=ANY($1::bigint[])", []int64{owner, actor, target})
			}()
			exec("INSERT INTO guilds(id,name,owner_id) VALUES($1,'race',$2)", guild, owner)
			for _, id := range []int64{owner, actor, target} {
				exec("INSERT INTO members(guild_id,user_id) VALUES($1,$2)", guild, id)
			}
			exec("INSERT INTO roles(id,guild_id,name,permissions,position) VALUES($1,$2,'moderator',$3,10),($4,$2,'higher',0,20)", low, guild, int64(permissions.PermBanMembers|permissions.PermKickMembers), high)
			exec("INSERT INTO member_roles(guild_id,user_id,role_id) VALUES($1,$2,$3)", guild, actor, low)
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(context.Background()) }()
			var pid int32
			if err = tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, "INSERT INTO member_roles(guild_id,user_id,role_id) VALUES($1,$2,$3)", guild, target, high); err != nil {
				t.Fatal(err)
			}
			guilds, members, roles := database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool)
			bans := database.NewBanRepository(pool)
			perms := NewPermissionChecker(guilds, members, roles, database.NewChannelOverrideRepository(pool))
			gw := &storageTestGateway{}
			call := func() error {
				if op == "ban" {
					return NewBanService(guilds, members, roles, bans, gw, perms).BanMember(ctx, guild, actor, target, nil)
				}
				return NewMemberService(members, guilds, roles, gw, perms).KickMember(ctx, guild, actor, target)
			}
			done := make(chan error, 1)
			go func() { done <- call() }()
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()
			for {
				var blocked bool
				if err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))", pid).Scan(&blocked); err != nil {
					t.Fatal(err)
				}
				if blocked {
					break
				}
				select {
				case err := <-done:
					t.Fatalf("moderation returned before assignment committed: %v", err)
				case <-ctx.Done():
					t.Fatal("did not observe moderation blocked by assignment")
				case <-ticker.C:
				}
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err = <-done; !errors.Is(err, ErrRoleHierarchy) {
				t.Fatalf("want role hierarchy rejection after committed assignment, got %v", err)
			}
			member, err := members.GetByGuildAndUser(ctx, guild, target)
			if err != nil || member == nil {
				t.Fatalf("target removed: %v", err)
			}
			// Removing the elevated role restores the adjacent allowed case.
			exec("DELETE FROM member_roles WHERE guild_id=$1 AND user_id=$2", guild, target)
			if err = call(); err != nil {
				t.Fatalf("allowed moderation: %v", err)
			}
		})
	}
}

func TestKickGuildDeletedWhileWaiting(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sf, _ := snowflake.NewGenerator(18, 20)
	owner, target, guild := sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []int64{owner, target} {
		exec("INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'kick','fixture')", id, sf.Generate().String())
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM guilds WHERE id=$1", guild)
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id=ANY($1::bigint[])", []int64{owner, target})
	}()
	exec("INSERT INTO guilds(id,name,owner_id) VALUES($1,'kick',$2)", guild, owner)
	for _, id := range []int64{owner, target} {
		exec("INSERT INTO members(guild_id,user_id) VALUES($1,$2)", guild, id)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Hold just the parent first, so preflight can still read committed membership.
	var pid int32
	if err = tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "SELECT id FROM guilds WHERE id=$1 FOR UPDATE", guild); err != nil {
		t.Fatal(err)
	}
	guilds, members, roles := database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool)
	gw := &storageTestGateway{}
	svc := NewMemberService(members, guilds, roles, gw, NewPermissionChecker(guilds, members, roles, database.NewChannelOverrideRepository(pool)))
	done := make(chan error, 1)
	go func() { done <- svc.KickMember(ctx, guild, owner, target) }()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))", pid).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("kick returned before parent release: %v", err)
		case <-ctx.Done():
			t.Fatal("kick did not wait on parent lock")
		case <-ticker.C:
		}
	}
	if _, err = tx.Exec(ctx, "DELETE FROM guilds WHERE id=$1", guild); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	err = <-done
	var se *ServiceError
	if !errors.As(err, &se) || !errors.Is(err, ErrNotFound) || se.Code != "UNKNOWN_GUILD" {
		t.Fatalf("want UNKNOWN_GUILD 404, got %v", err)
	}
	if len(gw.events) != 0 {
		t.Fatal("events emitted for deleted guild")
	}
}
