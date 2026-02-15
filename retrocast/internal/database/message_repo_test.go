package database

import (
	"context"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestMessageRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	repo := NewMessageRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)

	msg := &models.Message{
		ID:        nextID(),
		ChannelID: ch.ID,
		AuthorID:  owner.ID,
		Content:   "Hello, world!",
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, msg.ID) })

	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil after Create")
	}
	if got.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", got.Content, "Hello, world!")
	}
	if got.AuthorUsername != owner.Username {
		t.Errorf("AuthorUsername = %q, want %q", got.AuthorUsername, owner.Username)
	}
}

func TestMessageRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewMessageRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestMessageRepo_GetByChannelID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	repo := NewMessageRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)

	// Create 3 messages with ascending IDs
	var msgIDs []int64
	for i := 0; i < 3; i++ {
		msg := &models.Message{
			ID:        nextID(),
			ChannelID: ch.ID,
			AuthorID:  owner.ID,
			Content:   "Message " + string(rune('A'+i)),
			CreatedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("Create msg %d: %v", i, err)
		}
		msgIDs = append(msgIDs, msg.ID)
		msgID := msg.ID
		t.Cleanup(func() { _ = repo.Delete(ctx, msgID) })
	}

	// Get all messages (no cursor)
	messages, err := repo.GetByChannelID(ctx, ch.ID, nil, 100)
	if err != nil {
		t.Fatalf("GetByChannelID: %v", err)
	}
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(messages))
	}
	// Verify DESC ordering (newest first)
	if messages[0].ID < messages[len(messages)-1].ID {
		t.Error("messages not in DESC order")
	}
}

func TestMessageRepo_GetByChannelID_Pagination(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	repo := NewMessageRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)

	var msgIDs []int64
	for i := 0; i < 3; i++ {
		msg := &models.Message{
			ID:        nextID(),
			ChannelID: ch.ID,
			AuthorID:  owner.ID,
			Content:   "Paginated",
			CreatedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("Create: %v", err)
		}
		msgIDs = append(msgIDs, msg.ID)
		msgID := msg.ID
		t.Cleanup(func() { _ = repo.Delete(ctx, msgID) })
	}

	// Use the last (highest) message ID as cursor
	cursor := msgIDs[2]
	messages, err := repo.GetByChannelID(ctx, ch.ID, &cursor, 100)
	if err != nil {
		t.Fatalf("GetByChannelID with cursor: %v", err)
	}
	for _, m := range messages {
		if m.ID >= cursor {
			t.Errorf("message ID %d should be < cursor %d", m.ID, cursor)
		}
	}
}

func TestMessageRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	repo := NewMessageRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)

	msg := &models.Message{
		ID:        nextID(),
		ChannelID: ch.ID,
		AuthorID:  owner.ID,
		Content:   "Original",
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, msg.ID) })

	now := time.Now().Truncate(time.Microsecond)
	msg.Content = "Edited"
	msg.EditedAt = &now
	if err := repo.Update(ctx, msg); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Content != "Edited" {
		t.Errorf("Content = %q, want %q", got.Content, "Edited")
	}
	if got.EditedAt == nil {
		t.Error("EditedAt should not be nil after edit")
	}
}

func TestMessageRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	repo := NewMessageRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)

	msg := &models.Message{
		ID:        nextID(),
		ChannelID: ch.ID,
		AuthorID:  owner.ID,
		Content:   "To Delete",
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, msg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}

// createTestChannel inserts a channel and registers cleanup.
func createTestChannel(t *testing.T, repo ChannelRepository, guildID int64) *models.Channel {
	t.Helper()
	ctx := context.Background()
	id := nextID()
	ch := &models.Channel{
		ID:       id,
		GuildID:  guildID,
		Name:     "test-channel-" + time.Now().Format("150405.000000000"),
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("createTestChannel: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, id) })
	return ch
}
