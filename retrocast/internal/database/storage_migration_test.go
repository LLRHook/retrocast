package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStorageMigrationRoundTrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	schema := fmt.Sprintf("l15_migration_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	cfg := pool.Config().Copy()
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	isolated, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer isolated.Close()
	conn, err := isolated.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	migration := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	entries, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range entries {
		if filepath.Base(file) < "000020" {
			exec(migration(filepath.Base(file)))
		}
	}
	exec(`INSERT INTO users(id,username,display_name,password_hash) VALUES(1,'migration','Migration','fixture'); INSERT INTO guilds(id,name,owner_id) VALUES(2,'Migration',1); INSERT INTO channels(id,guild_id,name) VALUES(3,2,'Migration'); INSERT INTO messages(id,channel_id,author_id,content) VALUES(4,3,1,'original'); INSERT INTO attachments(id,message_id,filename,content_type,size,storage_key) VALUES(5,4,'original.txt','text/plain',1,'original'); INSERT INTO dm_channels(id,type) VALUES(6,1)`)
	up, down := migration("000020_dm_messages_and_pending_attachments.up.sql"), migration("000020_dm_messages_and_pending_attachments.down.sql")
	exec(up)
	checkOriginal := func() {
		t.Helper()
		var content string
		var uploader, channel int64
		if err := conn.QueryRow(ctx, `SELECT m.content,a.uploader_id,a.channel_id FROM messages m JOIN attachments a ON a.message_id=m.id WHERE m.id=4`).Scan(&content, &uploader, &channel); err != nil || content != "original" || uploader != 1 || channel != 3 {
			t.Fatalf("backfill/data preservation: %s %d %d %v", content, uploader, channel, err)
		}
	}
	checkOriginal()
	for _, kind := range []string{"DM", "pending"} {
		if kind == "DM" {
			exec(`INSERT INTO messages(id,channel_id,author_id,content) VALUES(7,6,1,'dm')`)
		} else {
			exec(`INSERT INTO attachments(id,message_id,uploader_id,channel_id,filename,content_type,size,storage_key) VALUES(8,NULL,1,3,'pending','text/plain',1,'pending')`)
		}
		if _, err := conn.Exec(ctx, down); err == nil {
			t.Fatal("unsafe down succeeded")
		}
		exec("ROLLBACK")
		checkOriginal()
		if kind == "DM" {
			exec("DELETE FROM messages WHERE id=7")
		} else {
			exec("DELETE FROM attachments WHERE id=8")
		}
	}
	exec(down)
	var content string
	if err := conn.QueryRow(ctx, "SELECT content FROM messages WHERE id=4").Scan(&content); err != nil || content != "original" {
		t.Fatal("down lost guild data")
	}
	exec(up)
	checkOriginal()
	// A genuinely empty database schema also migrates up/down/up.
	exec(`DELETE FROM guilds; DELETE FROM dm_channels; DELETE FROM users`)
	exec(down)
	exec(up)
}

func TestStorageParentIntegrityAndLocks(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	user := createTestUserSimple(t, NewUserRepository(pool))
	guild := createTestGuild(t, NewGuildRepository(pool), user.ID)
	ch := createTestChannel(t, NewChannelRepository(pool), guild.ID)
	for _, table := range []string{"channels", "dm_channels"} {
		t.Run(table, func(t *testing.T) {
			parent := ch.ID
			if table == "dm_channels" {
				parent = nextID()
				if _, err := pool.Exec(ctx, "INSERT INTO dm_channels(id,type) VALUES($1,1)", parent); err != nil {
					t.Fatal(err)
				}
				defer cleanupDM(t, pool, parent)
			}
			insert := `INSERT INTO messages(id,channel_id,author_id,content) VALUES($1,$2,$3,'locking')`
			// A message holds a key-share lock, so parent deletion must wait.
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err = tx.Exec(ctx, insert, nextID(), parent, user.ID); err != nil {
				t.Fatal(err)
			}
			blocked, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
			_, err = pool.Exec(blocked, "DELETE FROM "+table+" WHERE id=$1", parent)
			cancel()
			if err == nil {
				t.Fatal("parent delete bypassed message key-share lock")
			}
			if err = tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			// Parent deletion holds the row lock: a new message cannot race through.
			tx, err = pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err = tx.Exec(ctx, "DELETE FROM "+table+" WHERE id=$1", parent); err != nil {
				t.Fatal(err)
			}
			blocked, cancel = context.WithTimeout(ctx, 150*time.Millisecond)
			_, err = pool.Exec(blocked, insert, nextID(), parent, user.ID)
			cancel()
			if err == nil {
				t.Fatal("message insert bypassed parent deletion lock")
			}
			if err = tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
	repo := NewMessageRepository(pool)
	msg := createTestMessage(t, repo, ch.ID, user.ID)
	for _, query := range []string{`INSERT INTO messages(id,channel_id,author_id,content) VALUES($1,$2,$3,'invalid')`, `UPDATE messages SET channel_id=$2 WHERE id=$1 AND author_id=$3`} {
		_, err := pool.Exec(ctx, query, msg.ID, nextID(), user.ID)
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) || pgerr.Code != "23503" {
			t.Fatalf("unknown parent accepted or wrong error: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, "DELETE FROM guilds WHERE id=$1", guild.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetByID(ctx, msg.ID); err != nil || got != nil {
		t.Fatalf("guild cascade lost: %v %v", got, err)
	}
}
