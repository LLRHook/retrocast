package database

import (
	"context"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestBanRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	bannedUser := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	reason := "spamming"
	ban := &models.Ban{
		GuildID:   guild.ID,
		UserID:    bannedUser.ID,
		Reason:    &reason,
		CreatedBy: owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, ban); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, bannedUser.ID) })

	got, err := repo.GetByGuildAndUser(ctx, guild.ID, bannedUser.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got == nil {
		t.Fatal("GetByGuildAndUser returned nil after Create")
	}
	if got.Reason == nil || *got.Reason != "spamming" {
		t.Errorf("Reason = %v, want %q", got.Reason, "spamming")
	}
	if got.CreatedBy != owner.ID {
		t.Errorf("CreatedBy = %d, want %d", got.CreatedBy, owner.ID)
	}
}

func TestBanRepo_Create_Duplicate(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	bannedUser := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ban := &models.Ban{
		GuildID:   guild.ID,
		UserID:    bannedUser.ID,
		CreatedBy: owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, ban); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, bannedUser.ID) })

	err := repo.Create(ctx, ban)
	if err == nil {
		t.Fatal("expected error for duplicate ban, got nil")
	}
}

func TestBanRepo_GetByGuildAndUser_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByGuildAndUser(ctx, 999999999, 999999999)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestBanRepo_GetByGuildID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)
	user3 := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	for _, uid := range []int64{user2.ID, user3.ID} {
		ban := &models.Ban{
			GuildID:   guild.ID,
			UserID:    uid,
			CreatedBy: owner.ID,
			CreatedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(ctx, ban); err != nil {
			t.Fatalf("Create ban: %v", err)
		}
		userID := uid
		t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, userID) })
	}

	bans, err := repo.GetByGuildID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByGuildID: %v", err)
	}
	if len(bans) < 2 {
		t.Errorf("expected at least 2 bans, got %d", len(bans))
	}
}

func TestBanRepo_GetByGuildID_Empty(t *testing.T) {
	pool := testPool(t)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	bans, err := repo.GetByGuildID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByGuildID: %v", err)
	}
	if len(bans) != 0 {
		t.Errorf("expected empty slice, got %d", len(bans))
	}
}

func TestBanRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	bannedUser := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ban := &models.Ban{
		GuildID:   guild.ID,
		UserID:    bannedUser.ID,
		CreatedBy: owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, ban); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, guild.ID, bannedUser.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByGuildAndUser(ctx, guild.ID, bannedUser.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}

func TestBanRepo_Create_NilReason(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewBanRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	bannedUser := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	ban := &models.Ban{
		GuildID:   guild.ID,
		UserID:    bannedUser.ID,
		Reason:    nil,
		CreatedBy: owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, ban); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, bannedUser.ID) })

	got, err := repo.GetByGuildAndUser(ctx, guild.ID, bannedUser.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got == nil {
		t.Fatal("GetByGuildAndUser returned nil")
	}
	if got.Reason != nil {
		t.Errorf("Reason = %v, want nil", got.Reason)
	}
}
