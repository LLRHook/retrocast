package service

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

type storageTestEvent struct {
	user, guild int64
	kind        string
	data        any
}
type storageTestGateway struct {
	sync.Mutex
	events []storageTestEvent
}

func (g *storageTestGateway) DispatchToUser(id int64, event string, data any) {
	g.Lock()
	defer g.Unlock()
	g.events = append(g.events, storageTestEvent{user: id, kind: event, data: data})
}
func (g *storageTestGateway) DispatchToGuild(id int64, event string, data any) {
	g.Lock()
	defer g.Unlock()
	g.events = append(g.events, storageTestEvent{guild: id, kind: event, data: data})
}
func (g *storageTestGateway) DispatchToGuildExcept(id, except int64, event string, data any) {
	g.DispatchToGuild(id, event, data)
}
func (*storageTestGateway) SubscribeToGuild(int64, int64)     {}
func (*storageTestGateway) UnsubscribeFromGuild(int64, int64) {}

type storageTestFiles struct{ deleted int }

func (*storageTestFiles) Upload(context.Context, string, io.Reader, int64, string) error { return nil }
func (*storageTestFiles) GetURL(key string) string                                       { return "http://files/" + key }
func (f *storageTestFiles) Delete(context.Context, string) error                         { f.deleted++; return nil }

func TestStorageServices(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sf, _ := snowflake.NewGenerator(7, 7)
	user, other, outsider, guild, ch := sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64(), sf.Generate().Int64()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []int64{user, other, outsider} {
		exec(`INSERT INTO users(id,username,display_name,password_hash) VALUES($1,$2,'storage test','fixture')`, id, "storage-"+sf.Generate().String())
	}
	exec(`INSERT INTO guilds(id,name,owner_id) VALUES($1,'storage test',$2)`, guild, user)
	exec(`INSERT INTO channels(id,guild_id,name) VALUES($1,$2,'storage test')`, ch, guild)
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM guilds WHERE id=$1", guild)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id=ANY($1::bigint[])", []int64{user, other, outsider})
	}()
	messages, channels, dms := database.NewMessageRepository(pool), database.NewChannelRepository(pool), database.NewDMChannelRepository(pool)
	perms := NewPermissionChecker(database.NewGuildRepository(pool), database.NewMemberRepository(pool), database.NewRoleRepository(pool), database.NewChannelOverrideRepository(pool))
	gw := &storageTestGateway{}
	files := &storageTestFiles{}
	svc := NewMessageService(messages, channels, dms, sf, gw, perms)
	svc.SetFileStorage(files)
	for _, kind := range []models.DMChannelType{models.DMTypeDM, models.DMTypeGroupDM} {
		t.Run("dm"+string(rune('0'+kind)), func(t *testing.T) {
			dm := &models.DMChannel{ID: sf.Generate().Int64(), Type: kind, CreatedAt: time.Now(), Recipients: []models.User{{ID: user}, {ID: other}}}
			if err := dms.Create(ctx, dm); err != nil {
				t.Fatal(err)
			}
			defer func() { _, _ = pool.Exec(ctx, "DELETE FROM dm_channels WHERE id=$1", dm.ID) }()
			start := len(gw.events)
			msg, err := svc.SendMessage(ctx, dm.ID, user, "private storagesentinel")
			if err != nil {
				t.Fatal(err)
			}
			if len(gw.events) != start+2 {
				t.Fatal("DM must reach exactly its recipients")
			}
			for _, event := range gw.events[start:] {
				if event.guild != 0 || (event.user != user && event.user != other) || event.kind != gateway.EventMessageCreate {
					t.Fatalf("DM fanout: %+v", event)
				}
			}
			history, err := svc.GetMessages(ctx, dm.ID, other, nil, 50)
			if err != nil || len(history) != 1 || history[0].ID != msg.ID {
				t.Fatalf("history: %v %v", history, err)
			}
			if history[0].Attachments == nil {
				t.Fatal("attachments must be []")
			}
			if _, err = svc.SendMessage(ctx, dm.ID, outsider, "refused"); !errors.Is(err, ErrForbidden) {
				t.Fatalf("nonrecipient send: %v", err)
			}
			if _, err = svc.GetMessages(ctx, dm.ID, outsider, nil, 50); !errors.Is(err, ErrForbidden) {
				t.Fatalf("nonrecipient history: %v", err)
			}
			if _, err = svc.EditMessage(ctx, dm.ID, msg.ID, user, "edited private"); err != nil {
				t.Fatal(err)
			}
			reads := NewReadStateService(database.NewReadStateRepository(pool), channels, dms, perms)
			if err = reads.Ack(ctx, dm.ID, msg.ID, other); err != nil {
				t.Fatal(err)
			}
			reactions := NewReactionService(database.NewReactionRepository(pool), messages, channels, dms, gw, perms)
			if err = reactions.AddReaction(ctx, dm.ID, msg.ID, other, "ok"); err != nil {
				t.Fatal(err)
			}
			if ids, err := reactions.GetReactions(ctx, dm.ID, msg.ID, other, "ok", 25); err != nil || len(ids) != 1 {
				t.Fatalf("reactions %v %v", ids, err)
			}
			if err = svc.DeleteMessage(ctx, dm.ID, msg.ID, other); !errors.Is(err, ErrForbidden) {
				t.Fatalf("non-author delete: %v", err)
			}
			if err = svc.DeleteMessage(ctx, dm.ID, msg.ID, user); err != nil {
				t.Fatal(err)
			}
			if _, err = svc.SendMessage(ctx, dm.ID, user, "private storagesentinel"); err != nil {
				t.Fatal(err)
			}
			if err = dms.RemoveRecipient(ctx, dm.ID, other); err != nil {
				t.Fatal(err)
			}
			if _, err = svc.SendMessage(ctx, dm.ID, other, "removed"); !errors.Is(err, ErrForbidden) {
				t.Fatalf("removed recipient: %v", err)
			}
			found, err := messages.SearchMessages(ctx, guild, "storagesentinel", nil, nil, nil, 25)
			if err != nil || len(found) != 0 {
				t.Fatalf("DM leaked to search: %v %v", found, err)
			}
		})
	}
	upload := NewUploadService(database.NewAttachmentRepository(pool), channels, sf, files, perms)
	pending := func() *models.Attachment {
		t.Helper()
		a, err := upload.UploadFile(ctx, ch, user, "proof.txt", 5, "text/plain", strings.NewReader("proof"))
		if err != nil {
			t.Fatal(err)
		}
		if a.MessageID != nil {
			t.Fatal("upload is not pending")
		}
		return a
	}
	a := pending()
	msg, err := svc.SendMessage(ctx, ch, user, "guild storagesentinel", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertAttachment := func(m models.MessageWithAuthor) {
		t.Helper()
		if len(m.Attachments) != 1 || m.Attachments[0].ID != a.ID || m.Attachments[0].MessageID == nil || *m.Attachments[0].MessageID != msg.ID || !strings.HasPrefix(m.Attachments[0].URL, "http://files/") {
			t.Fatalf("missing attachment: %+v", m)
		}
	}
	assertAttachment(*msg)
	single, err := svc.GetMessage(ctx, ch, msg.ID, user)
	if err != nil {
		t.Fatal(err)
	}
	assertAttachment(*single)
	history, err := svc.GetMessages(ctx, ch, user, nil, 50)
	if err != nil || len(history) != 1 {
		t.Fatalf("history: %v %v", history, err)
	}
	assertAttachment(history[0])
	edited, err := svc.EditMessage(ctx, ch, msg.ID, user, "edited storagesentinel")
	if err != nil {
		t.Fatal(err)
	}
	assertAttachment(*edited)
	search := NewSearchService(messages, database.NewMemberRepository(pool), perms)
	search.SetFileStorage(files)
	exec(`INSERT INTO members(guild_id,user_id) VALUES($1,$2)`, guild, user)
	results, err := search.SearchMessages(ctx, guild, user, "storagesentinel", nil, nil, nil, 25)
	if err != nil || len(results) != 1 {
		t.Fatalf("search %v %v", results, err)
	}
	assertAttachment(results[0])
	count := func() int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM messages WHERE channel_id=$1", ch).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	b := pending()
	foreign := pending()
	exec(`UPDATE attachments SET uploader_id=$1 WHERE id=$2`, other, foreign.ID)
	otherCh := sf.Generate().Int64()
	exec(`INSERT INTO channels(id,guild_id,name) VALUES($1,$2,'other')`, otherCh, guild)
	cross := pending()
	exec(`UPDATE attachments SET channel_id=$1 WHERE id=$2`, otherCh, cross.ID)
	for _, ids := range [][]int64{{sf.Generate().Int64()}, {b.ID, b.ID}, {foreign.ID}, {cross.ID}, {a.ID}, {b.ID, a.ID}} {
		before, events := count(), len(gw.events)
		_, err := svc.SendMessage(ctx, ch, user, "must rollback", ids...)
		var se *ServiceError
		if !errors.As(err, &se) || se.Code != "INVALID_ATTACHMENTS" {
			t.Fatalf("generic refusal: %v", err)
		}
		if count() != before || len(gw.events) != events {
			t.Fatal("invalid attachments wrote a message or dispatched")
		}
	}
	var linked *int64
	if err = pool.QueryRow(ctx, "SELECT message_id FROM attachments WHERE id=$1", b.ID).Scan(&linked); err != nil || linked != nil {
		t.Fatalf("partial claim persisted: %v %v", linked, err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := svc.SendMessage(ctx, ch, user, "concurrent", b.ID); errs <- err })
	}
	wg.Wait()
	close(errs)
	successes, refusals := 0, 0
	for err := range errs {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrBadRequest) {
			refusals++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || refusals != 1 {
		t.Fatalf("concurrent claims: %d successes %d refusals", successes, refusals)
	}
	if err = svc.DeleteMessage(ctx, ch, msg.ID, user); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM attachments WHERE id=$1", a.ID).Scan(&n); err != nil || n != 0 {
		t.Fatal("attachment cascade failed")
	}
	// Pending uploads remain until their channel is removed; no object GC is promised.
}
