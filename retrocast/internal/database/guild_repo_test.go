package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestGuildRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewGuildRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)

	guild := &models.Guild{
		ID:        nextID(),
		Name:      "Test Guild",
		OwnerID:   owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, guild); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID) })

	got, err := repo.GetByID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil after Create")
	}
	if got.Name != guild.Name {
		t.Errorf("Name = %q, want %q", got.Name, guild.Name)
	}
	if got.OwnerID != guild.OwnerID {
		t.Errorf("OwnerID = %d, want %d", got.OwnerID, guild.OwnerID)
	}
}

func TestGuildRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewGuildRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent ID, got %+v", got)
	}
}

func TestGuildRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewGuildRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := &models.Guild{
		ID:        nextID(),
		Name:      "Before Update",
		OwnerID:   owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, guild); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID) })

	guild.Name = "After Update"
	if err := repo.Update(ctx, guild); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "After Update" {
		t.Errorf("Name = %q, want %q", got.Name, "After Update")
	}
}

func TestGuildRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewGuildRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := &models.Guild{
		ID:        nextID(),
		Name:      "To Delete",
		OwnerID:   owner.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, guild); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, guild.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByID after Delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}

func TestGuildRepo_GetByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	memberRepo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	member := &models.Member{
		GuildID:  guild.ID,
		UserID:   owner.ID,
		JoinedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := memberRepo.Create(ctx, member); err != nil {
		t.Fatalf("Create member: %v", err)
	}
	t.Cleanup(func() { _ = memberRepo.Delete(ctx, guild.ID, owner.ID) })

	guilds, err := guildRepo.GetByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}

	found := false
	for _, g := range guilds {
		if g.ID == guild.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetByUserID did not return the expected guild %d", guild.ID)
	}
}

func TestGuildRepo_GetByUserID_NoGuilds(t *testing.T) {
	pool := testPool(t)
	guildRepo := NewGuildRepository(pool)
	ctx := context.Background()

	guilds, err := guildRepo.GetByUserID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if len(guilds) != 0 {
		t.Errorf("expected empty slice, got %d guilds", len(guilds))
	}
}

// createTestUserSimple inserts a user using the UserRepository directly.
func createTestUserSimple(t *testing.T, repo UserRepository) *models.User {
	t.Helper()
	ctx := context.Background()
	id := nextID()
	user := &models.User{
		ID:           id,
		Username:     fmt.Sprintf("testuser_%d", id),
		DisplayName:  "Test",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("createTestUserSimple: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, user.ID) })
	return user
}
