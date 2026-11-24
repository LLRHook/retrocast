package database

import (
	"context"
	"github.com/victorivanov/retrocast/internal/models"
	"testing"
	"time"
)

func TestStorageDMParent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	user := createTestUserSimple(t, NewUserRepository(pool))
	dm := &models.DMChannel{ID: nextID(), Type: models.DMTypeDM, CreatedAt: time.Now(), Recipients: []models.User{{ID: user.ID}}}
	if err := NewDMChannelRepository(pool).Create(ctx, dm); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dm.ID) })
	repo := NewMessageRepository(pool)
	msg := createTestMessage(t, repo, dm.ID, user.ID)
	history, err := repo.GetByChannelID(ctx, dm.ID, nil, 50)
	if err != nil || len(history) != 1 || history[0].ID != msg.ID {
		t.Fatalf("DM history: %v %v", history, err)
	}
	if _, err = pool.Exec(ctx, "DELETE FROM dm_channels WHERE id=$1", dm.ID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil || got != nil {
		t.Fatalf("DM delete cascade: %v %v", got, err)
	}
}

func TestStoragePendingUpload(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	user := createTestUserSimple(t, NewUserRepository(pool))
	guild := createTestGuild(t, NewGuildRepository(pool), user.ID)
	ch := createTestChannel(t, NewChannelRepository(pool), guild.ID)
	_, err := pool.Exec(ctx, `INSERT INTO attachments(id,message_id,uploader_id,channel_id,filename,content_type,size,storage_key) VALUES($1,NULL,$2,$3,'pending.txt','text/plain',1,'pending')`, nextID(), user.ID, ch.ID)
	if err != nil {
		t.Fatalf("pending upload must be storable: %v", err)
	}
}
