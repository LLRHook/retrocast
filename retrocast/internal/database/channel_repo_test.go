package database

import (
	"context"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestChannelRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ch := &models.Channel{
		ID:       nextID(),
		GuildID:  guild.ID,
		Name:     "general",
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, ch.ID) })

	got, err := repo.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil after Create")
	}
	if got.Name != "general" {
		t.Errorf("Name = %q, want %q", got.Name, "general")
	}
	if got.GuildID != guild.ID {
		t.Errorf("GuildID = %d, want %d", got.GuildID, guild.ID)
	}
}

func TestChannelRepo_Create_DuplicateName(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ch1 := &models.Channel{
		ID:       nextID(),
		GuildID:  guild.ID,
		Name:     "duplicate-name",
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := repo.Create(ctx, ch1); err != nil {
		t.Fatalf("Create ch1: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, ch1.ID) })

	ch2 := &models.Channel{
		ID:       nextID(),
		GuildID:  guild.ID,
		Name:     "duplicate-name",
		Type:     models.ChannelTypeText,
		Position: 1,
	}
	err := repo.Create(ctx, ch2)
	if err == nil {
		t.Cleanup(func() { _ = repo.Delete(ctx, ch2.ID) })
		t.Fatal("expected error for duplicate channel name in same guild, got nil")
	}
}

func TestChannelRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestChannelRepo_GetByGuildID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ch1 := &models.Channel{ID: nextID(), GuildID: guild.ID, Name: "alpha", Type: models.ChannelTypeText, Position: 1}
	ch2 := &models.Channel{ID: nextID(), GuildID: guild.ID, Name: "beta", Type: models.ChannelTypeText, Position: 0}
	for _, ch := range []*models.Channel{ch1, ch2} {
		if err := repo.Create(ctx, ch); err != nil {
			t.Fatalf("Create %s: %v", ch.Name, err)
		}
		chID := ch.ID
		t.Cleanup(func() { _ = repo.Delete(ctx, chID) })
	}

	channels, err := repo.GetByGuildID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByGuildID: %v", err)
	}
	if len(channels) < 2 {
		t.Fatalf("expected at least 2 channels, got %d", len(channels))
	}
	// Verify ordering by position
	if channels[0].Position > channels[1].Position {
		t.Errorf("channels not ordered by position: %d > %d", channels[0].Position, channels[1].Position)
	}
}

func TestChannelRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ch := &models.Channel{
		ID:       nextID(),
		GuildID:  guild.ID,
		Name:     "before-update",
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, ch.ID) })

	topic := "Updated topic"
	ch.Name = "after-update"
	ch.Topic = &topic
	if err := repo.Update(ctx, ch); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "after-update" {
		t.Errorf("Name = %q, want %q", got.Name, "after-update")
	}
	if got.Topic == nil || *got.Topic != topic {
		t.Errorf("Topic = %v, want %q", got.Topic, topic)
	}
}

func TestChannelRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ch := &models.Channel{
		ID:       nextID(),
		GuildID:  guild.ID,
		Name:     "to-delete",
		Type:     models.ChannelTypeText,
		Position: 0,
	}
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, ch.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}

// createTestGuild inserts a guild and registers cleanup.
func createTestGuild(t *testing.T, repo GuildRepository, ownerID int64) *models.Guild {
	t.Helper()
	ctx := context.Background()
	guild := &models.Guild{
		ID:        nextID(),
		Name:      "TestGuild",
		OwnerID:   ownerID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, guild); err != nil {
		t.Fatalf("createTestGuild: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID) })
	return guild
}
