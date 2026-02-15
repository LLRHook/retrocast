package database

import (
	"context"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestChannelOverrideRepo_Set(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	roleRepo := NewRoleRepository(pool)
	repo := NewChannelOverrideRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)
	role := createTestRole(t, roleRepo, guild.ID)

	override := &models.ChannelOverride{
		ChannelID: ch.ID,
		RoleID:    role.ID,
		Allow:     0x10,
		Deny:      0x20,
	}
	if err := repo.Set(ctx, override); err != nil {
		t.Fatalf("Set: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, ch.ID, role.ID) })

	overrides, err := repo.GetByChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByChannel: %v", err)
	}
	if len(overrides) == 0 {
		t.Fatal("GetByChannel returned empty after Set")
	}

	found := false
	for _, o := range overrides {
		if o.RoleID == role.ID {
			found = true
			if o.Allow != 0x10 {
				t.Errorf("Allow = %d, want %d", o.Allow, 0x10)
			}
			if o.Deny != 0x20 {
				t.Errorf("Deny = %d, want %d", o.Deny, 0x20)
			}
		}
	}
	if !found {
		t.Error("override for role not found")
	}
}

func TestChannelOverrideRepo_Set_Upsert(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	roleRepo := NewRoleRepository(pool)
	repo := NewChannelOverrideRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)
	role := createTestRole(t, roleRepo, guild.ID)

	override := &models.ChannelOverride{
		ChannelID: ch.ID,
		RoleID:    role.ID,
		Allow:     0x10,
		Deny:      0x20,
	}
	if err := repo.Set(ctx, override); err != nil {
		t.Fatalf("Set initial: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, ch.ID, role.ID) })

	// Update via Set (upsert)
	override.Allow = 0x30
	override.Deny = 0x40
	if err := repo.Set(ctx, override); err != nil {
		t.Fatalf("Set upsert: %v", err)
	}

	overrides, err := repo.GetByChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByChannel: %v", err)
	}

	for _, o := range overrides {
		if o.RoleID == role.ID {
			if o.Allow != 0x30 {
				t.Errorf("Allow = %d, want %d", o.Allow, 0x30)
			}
			if o.Deny != 0x40 {
				t.Errorf("Deny = %d, want %d", o.Deny, 0x40)
			}
		}
	}
}

func TestChannelOverrideRepo_GetByChannel_Empty(t *testing.T) {
	pool := testPool(t)
	repo := NewChannelOverrideRepository(pool)
	ctx := context.Background()

	overrides, err := repo.GetByChannel(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByChannel: %v", err)
	}
	if len(overrides) != 0 {
		t.Errorf("expected empty slice, got %d", len(overrides))
	}
}

func TestChannelOverrideRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	channelRepo := NewChannelRepository(pool)
	roleRepo := NewRoleRepository(pool)
	repo := NewChannelOverrideRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	ch := createTestChannel(t, channelRepo, guild.ID)
	role := createTestRole(t, roleRepo, guild.ID)

	override := &models.ChannelOverride{
		ChannelID: ch.ID,
		RoleID:    role.ID,
		Allow:     0x10,
		Deny:      0x20,
	}
	if err := repo.Set(ctx, override); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := repo.Delete(ctx, ch.ID, role.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	overrides, err := repo.GetByChannel(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetByChannel: %v", err)
	}
	for _, o := range overrides {
		if o.RoleID == role.ID {
			t.Error("override still present after Delete")
		}
	}
}

// createTestRole inserts a role and registers cleanup.
func createTestRole(t *testing.T, repo RoleRepository, guildID int64) *models.Role {
	t.Helper()
	ctx := context.Background()
	role := &models.Role{
		ID:      nextID(),
		GuildID: guildID,
		Name:    "TestRole",
	}
	if err := repo.Create(ctx, role); err != nil {
		t.Fatalf("createTestRole: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, role.ID) })
	return role
}
