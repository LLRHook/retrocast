package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestInviteRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewInviteRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	code := fmt.Sprintf("T%07d", nextID()%10000000)
	invite := &models.Invite{
		Code:      code,
		GuildID:   guild.ID,
		CreatorID: owner.ID,
		MaxUses:   10,
		Uses:      0,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, invite); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, code) })

	got, err := repo.GetByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got == nil {
		t.Fatal("GetByCode returned nil after Create")
	}
	if got.GuildID != guild.ID {
		t.Errorf("GuildID = %d, want %d", got.GuildID, guild.ID)
	}
	if got.MaxUses != 10 {
		t.Errorf("MaxUses = %d, want 10", got.MaxUses)
	}
}

func TestInviteRepo_Create_DuplicateCode(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewInviteRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	code := fmt.Sprintf("D%07d", nextID()%10000000)
	invite := &models.Invite{
		Code:      code,
		GuildID:   guild.ID,
		CreatorID: owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, invite); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, code) })

	err := repo.Create(ctx, invite)
	if err == nil {
		t.Fatal("expected error for duplicate code, got nil")
	}
}

func TestInviteRepo_GetByCode_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewInviteRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByCode(ctx, "ZZZZZZZZ")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestInviteRepo_GetByGuildID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewInviteRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	for i := 0; i < 2; i++ {
		code := fmt.Sprintf("G%07d", nextID()%10000000)
		invite := &models.Invite{
			Code:      code,
			GuildID:   guild.ID,
			CreatorID: owner.ID,
			CreatedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(ctx, invite); err != nil {
			t.Fatalf("Create invite %d: %v", i, err)
		}
		c := code
		t.Cleanup(func() { _ = repo.Delete(ctx, c) })
	}

	invites, err := repo.GetByGuildID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByGuildID: %v", err)
	}
	if len(invites) < 2 {
		t.Errorf("expected at least 2 invites, got %d", len(invites))
	}
}

func TestInviteRepo_IncrementUses(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewInviteRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	code := fmt.Sprintf("I%07d", nextID()%10000000)
	invite := &models.Invite{
		Code:      code,
		GuildID:   guild.ID,
		CreatorID: owner.ID,
		Uses:      0,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, invite); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, code) })

	if err := repo.IncrementUses(ctx, code); err != nil {
		t.Fatalf("IncrementUses: %v", err)
	}

	got, err := repo.GetByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.Uses != 1 {
		t.Errorf("Uses = %d, want 1", got.Uses)
	}
}

func TestInviteRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewInviteRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	code := fmt.Sprintf("X%07d", nextID()%10000000)
	invite := &models.Invite{
		Code:      code,
		GuildID:   guild.ID,
		CreatorID: owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, invite); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, code); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}
