package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/snowflake"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestRoleReorderConcurrentAndRollback(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sf, _ := snowflake.NewGenerator(8, 8)
	owner, guild, a, b, def := sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'reorder','fixture')`, owner, "reorder-"+sf.Generate().String())
	manager1, manager2, managerRole := sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64()
	for _, id := range []int64{manager1, manager2} {
		exec(`INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'manager','fixture')`, id, "manager-"+sf.Generate().String())
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=ANY($1::bigint[])", []int64{manager1, manager2})
	}()
	exec(`INSERT INTO guilds(id,name,owner_id) VALUES($1,'reorder',$2)`, guild, owner)
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM guilds WHERE id=$1", guild)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=$1", owner)
	}()
	exec(`INSERT INTO roles(id,guild_id,name,position,is_default) VALUES($1,$4,'a',1,false),($2,$4,'b',2,false),($3,$4,'@everyone',0,true)`, a, b, def, guild)
	exec(`INSERT INTO roles(id,guild_id,name,position,permissions) VALUES($1,$2,'manager',10,16)`, managerRole, guild)
	for _, id := range []int64{manager1, manager2} {
		exec(`INSERT INTO members(guild_id,user_id) VALUES($1,$2)`, guild, id)
		exec(`INSERT INTO member_roles(guild_id,user_id,role_id) VALUES($1,$2,$3)`, guild, id, managerRole)
	}
	roles := database.NewRoleRepository(pool)
	guilds := database.NewGuildRepository(pool)
	members := database.NewMemberRepository(pool)
	overrides := database.NewChannelOverrideRepository(pool)
	gw := &storageTestGateway{}
	svc := NewRoleService(guilds, roles, members, database.NewChannelRepository(pool), overrides, sf, gw, NewPermissionChecker(guilds, members, roles, overrides))
	expected := []RolePosition{{ID: managerRole, Position: 10}, {ID: a, Position: 1}, {ID: b, Position: 2}, {ID: def, Position: 0}}
	order := []string{strconv.FormatInt(b, 10), strconv.FormatInt(a, 10), strconv.FormatInt(managerRole, 10)}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	start := make(chan struct{})
	for _, actor := range []int64{manager1, manager2} {
		wg.Add(1)
		go func(actor int64) {
			defer wg.Done()
			<-start
			_, err := svc.ReorderRoles(ctx, guild, actor, expected, order)
			results <- err
		}(actor)
	}
	close(start)
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
	got, _ := roles.GetByID(ctx, a)
	if got.Position != 2 {
		t.Fatal("not persisted")
	}
	if len(gw.events) != 2 {
		t.Fatalf("events=%d", len(gw.events))
	}
	// A database failure halfway through must roll back the first update and emit no events.
	exec(`CREATE FUNCTION l34_reorder_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='b' THEN RAISE EXCEPTION 'fixture failure'; END IF; RETURN NEW; END $$`)
	exec(fmt.Sprintf(`CREATE TRIGGER l34_reorder_fail BEFORE UPDATE ON roles FOR EACH ROW WHEN (NEW.guild_id=%d) EXECUTE FUNCTION l34_reorder_fail()`, guild))
	defer func() {
		_, _ = pool.Exec(ctx, `DROP TRIGGER IF EXISTS l34_reorder_fail ON roles; DROP FUNCTION IF EXISTS l34_reorder_fail()`)
	}()
	_, err = svc.ReorderRoles(ctx, guild, owner, []RolePosition{{ID: managerRole, Position: 10}, {ID: a, Position: 2}, {ID: b, Position: 1}, {ID: def, Position: 0}}, []string{strconv.FormatInt(a, 10), strconv.FormatInt(b, 10), strconv.FormatInt(managerRole, 10)})
	if err == nil || !strings.Contains(err.Error(), "fixture failure") {
		t.Fatalf("expected injected failure, got %v", err)
	}
	got, _ = roles.GetByID(ctx, a)
	if got.Position != 2 || len(gw.events) != 2 {
		t.Fatal("partial write or event on rollback")
	}
}
