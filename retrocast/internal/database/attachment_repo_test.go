package database

import (
	"context"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestAttachmentRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	messageRepo := NewMessageRepository(pool)
	repo := NewAttachmentRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)
	msg := createTestMessage(t, messageRepo, ch.ID, owner.ID)

	att := &models.Attachment{
		ID:          nextID(),
		MessageID:   msg.ID,
		Filename:    "image.png",
		ContentType: "image/png",
		Size:        12345,
		StorageKey:  "uploads/test/image.png",
	}
	if err := repo.Create(ctx, att); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, att.ID) })

	attachments, err := repo.GetByMessageID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByMessageID: %v", err)
	}
	if len(attachments) == 0 {
		t.Fatal("GetByMessageID returned empty after Create")
	}

	got := attachments[0]
	if got.Filename != "image.png" {
		t.Errorf("Filename = %q, want %q", got.Filename, "image.png")
	}
	if got.ContentType != "image/png" {
		t.Errorf("ContentType = %q, want %q", got.ContentType, "image/png")
	}
	if got.Size != 12345 {
		t.Errorf("Size = %d, want 12345", got.Size)
	}
}

func TestAttachmentRepo_GetByMessageID_Empty(t *testing.T) {
	pool := testPool(t)
	repo := NewAttachmentRepository(pool)
	ctx := context.Background()

	attachments, err := repo.GetByMessageID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByMessageID: %v", err)
	}
	if len(attachments) != 0 {
		t.Errorf("expected empty slice, got %d", len(attachments))
	}
}

func TestAttachmentRepo_GetByMessageID_Multiple(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	messageRepo := NewMessageRepository(pool)
	repo := NewAttachmentRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)
	msg := createTestMessage(t, messageRepo, ch.ID, owner.ID)

	for i := 0; i < 3; i++ {
		att := &models.Attachment{
			ID:          nextID(),
			MessageID:   msg.ID,
			Filename:    "file" + string(rune('0'+i)) + ".txt",
			ContentType: "text/plain",
			Size:        int64(100 + i),
			StorageKey:  "uploads/test/file" + string(rune('0'+i)),
		}
		if err := repo.Create(ctx, att); err != nil {
			t.Fatalf("Create attachment %d: %v", i, err)
		}
		attID := att.ID
		t.Cleanup(func() { _ = repo.Delete(ctx, attID) })
	}

	attachments, err := repo.GetByMessageID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByMessageID: %v", err)
	}
	if len(attachments) < 3 {
		t.Errorf("expected at least 3 attachments, got %d", len(attachments))
	}
}

func TestAttachmentRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	messageRepo := NewMessageRepository(pool)
	repo := NewAttachmentRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)
	msg := createTestMessage(t, messageRepo, ch.ID, owner.ID)

	att := &models.Attachment{
		ID:          nextID(),
		MessageID:   msg.ID,
		Filename:    "to-delete.txt",
		ContentType: "text/plain",
		Size:        100,
		StorageKey:  "uploads/test/to-delete",
	}
	if err := repo.Create(ctx, att); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, att.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	attachments, err := repo.GetByMessageID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByMessageID: %v", err)
	}
	for _, a := range attachments {
		if a.ID == att.ID {
			t.Error("attachment still present after Delete")
		}
	}
}

// createTestMessage inserts a message and registers cleanup.
func createTestMessage(t *testing.T, repo MessageRepository, channelID, authorID int64) *models.Message {
	t.Helper()
	ctx := context.Background()
	msg := &models.Message{
		ID:        nextID(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   "test message",
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("createTestMessage: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, msg.ID) })
	return msg
}
