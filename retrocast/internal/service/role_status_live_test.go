package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
	"os"
	"testing"
	"time"
)

func TestRoleMutationGuildStatus(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sf, _ := snowflake.NewGenerator(9, 8)
	owner, outsider, guild, missing := sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []int64{owner, outsider} {
		exec(`INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'status','fixture')`, id, "status-"+sf.Generate().String())
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM guilds WHERE id=$1", guild)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=ANY($1::bigint[])", []int64{owner, outsider})
	}()
	exec(`INSERT INTO guilds(id,name,owner_id) VALUES($1,'status',$2)`, guild, owner)
	roles, guilds, members, overrides := database.NewRoleRepository(pool), database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewChannelOverrideRepository(pool)
	svc := NewRoleService(guilds, roles, members, database.NewChannelRepository(pool), overrides, sf, &storageTestGateway{}, NewPermissionChecker(guilds, members, roles, overrides))
	call := func(ctx context.Context, op string, g, actor int64) error {
		switch op {
		case "create":
			_, err := svc.CreateRole(ctx, g, actor, "role", 0, 0, 1)
			return err
		case "update":
			_, err := svc.UpdateRole(ctx, g, actor, 1, nil, nil, nil, nil)
			return err
		case "delete":
			return svc.DeleteRole(ctx, g, actor, 1)
		default:
			_, err := svc.ReorderRoles(ctx, g, actor, []RolePosition{}, []string{})
			return err
		}
	}
	for _, op := range []string{"create", "update", "delete", "reorder"} {
		t.Run("deleted_between_preflight_and_lock/"+op, func(t *testing.T) {
			hook := &deletingGuildRepo{GuildRepository: guilds, beforeReturn: func() { exec("DELETE FROM guilds WHERE id=$1", guild) }}
			original := svc
			svc = NewRoleService(hook, roles, members, database.NewChannelRepository(pool), overrides, sf, &storageTestGateway{}, NewPermissionChecker(hook, members, roles, overrides))
			err := call(ctx, op, guild, owner)
			svc = original
			var se *ServiceError
			if !errors.As(err, &se) || !errors.Is(err, ErrNotFound) || se.Code != "NOT_FOUND" || se.Message != "guild not found" {
				t.Fatalf("want 404 NOT_FOUND guild not found, got %v", err)
			}
			exec(`INSERT INTO guilds(id,name,owner_id) VALUES($1,'status',$2)`, guild, owner)
		})
	}

	for _, op := range []string{"create", "update", "delete", "reorder"} {
		t.Run("missing/"+op, func(t *testing.T) {
			err := call(ctx, op, missing, owner)
			var se *ServiceError
			if !errors.As(err, &se) || !errors.Is(err, ErrNotFound) || se.Code != "NOT_FOUND" || se.Message != "guild not found" {
				t.Fatalf("want 404 NOT_FOUND guild not found, got %v", err)
			}
		})
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT id FROM guilds WHERE id=$1 FOR UPDATE", guild); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"create", "update", "delete", "reorder"} {
		t.Run("outsider_before_lock/"+op, func(t *testing.T) {
			deadline, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
			defer cancel()
			err := call(deadline, op, guild, outsider)
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("want immediate 403 without lock, got %v", err)
			}
		})
	}
	_ = tx.Rollback(ctx)
	exec("DELETE FROM guilds WHERE id=$1", guild)
	for _, op := range []string{"create", "update", "delete", "reorder"} {
		t.Run("deleted/"+op, func(t *testing.T) {
			err := call(ctx, op, guild, owner)
			var se *ServiceError
			if !errors.As(err, &se) || !errors.Is(err, ErrNotFound) || se.Code != "NOT_FOUND" || se.Message != "guild not found" {
				t.Fatalf("want 404 NOT_FOUND guild not found, got %v", err)
			}
		})
	}
}

type deletingGuildRepo struct {
	database.GuildRepository
	beforeReturn func()
}

func (g *deletingGuildRepo) GetByID(ctx context.Context, id int64) (*models.Guild, error) {
	row, err := g.GuildRepository.GetByID(ctx, id)
	if g.beforeReturn != nil {
		fn := g.beforeReturn
		g.beforeReturn = nil
		fn()
	}
	return row, err
}
