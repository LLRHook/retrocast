package database

import (
	"context"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestRoleRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewRoleRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	role := &models.Role{
		ID:          nextID(),
		GuildID:     guild.ID,
		Name:        "Moderator",
		Color:       0xFF0000,
		Permissions: 0x8,
		Position:    1,
		IsDefault:   false,
	}
	if err := repo.Create(ctx, role); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, role.ID) })

	got, err := repo.GetByID(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil after Create")
	}
	if got.Name != "Moderator" {
		t.Errorf("Name = %q, want %q", got.Name, "Moderator")
	}
	if got.Color != 0xFF0000 {
		t.Errorf("Color = %d, want %d", got.Color, 0xFF0000)
	}
	if got.Permissions != 0x8 {
		t.Errorf("Permissions = %d, want %d", got.Permissions, 0x8)
	}
}

func TestRoleRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewRoleRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestRoleRepo_GetByGuildID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewRoleRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	role1 := &models.Role{ID: nextID(), GuildID: guild.ID, Name: "Admin", Position: 2}
	role2 := &models.Role{ID: nextID(), GuildID: guild.ID, Name: "Member", Position: 1, IsDefault: true}
	for _, r := range []*models.Role{role1, role2} {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.Name, err)
		}
		rID := r.ID
		t.Cleanup(func() { _ = repo.Delete(ctx, rID) })
	}

	roles, err := repo.GetByGuildID(ctx, guild.ID)
	if err != nil {
		t.Fatalf("GetByGuildID: %v", err)
	}
	if len(roles) < 2 {
		t.Fatalf("expected at least 2 roles, got %d", len(roles))
	}
	// Verify ordering by position
	if roles[0].Position > roles[1].Position {
		t.Errorf("roles not ordered by position: %d > %d", roles[0].Position, roles[1].Position)
	}
}

func TestRoleRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewRoleRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	role := &models.Role{
		ID:          nextID(),
		GuildID:     guild.ID,
		Name:        "Before",
		Color:       0,
		Permissions: 0,
		Position:    0,
	}
	if err := repo.Create(ctx, role); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, role.ID) })

	role.Name = "After"
	role.Color = 0x00FF00
	role.Permissions = 0x10
	if err := repo.Update(ctx, role); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "After" {
		t.Errorf("Name = %q, want %q", got.Name, "After")
	}
	if got.Color != 0x00FF00 {
		t.Errorf("Color = %d, want %d", got.Color, 0x00FF00)
	}
}

func TestRoleRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewRoleRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	role := &models.Role{
		ID:      nextID(),
		GuildID: guild.ID,
		Name:    "ToDelete",
	}
	if err := repo.Create(ctx, role); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, role.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}

func TestRoleRepo_GetByMember(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	roleRepo := NewRoleRepository(pool)
	memberRepo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	role := &models.Role{
		ID:       nextID(),
		GuildID:  guild.ID,
		Name:     "TestRole",
		Position: 1,
	}
	if err := roleRepo.Create(ctx, role); err != nil {
		t.Fatalf("Create role: %v", err)
	}
	t.Cleanup(func() { _ = roleRepo.Delete(ctx, role.ID) })

	member := createTestMember(t, memberRepo, guild.ID, owner.ID)
	_ = member

	if err := memberRepo.AddRole(ctx, guild.ID, owner.ID, role.ID); err != nil {
		t.Fatalf("AddRole: %v", err)
	}
	t.Cleanup(func() { _ = memberRepo.RemoveRole(ctx, guild.ID, owner.ID, role.ID) })

	roles, err := roleRepo.GetByMember(ctx, guild.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetByMember: %v", err)
	}

	found := false
	for _, r := range roles {
		if r.ID == role.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetByMember did not return assigned role %d", role.ID)
	}
}

func TestRoleRepo_GetByMember_NoRoles(t *testing.T) {
	pool := testPool(t)
	roleRepo := NewRoleRepository(pool)
	ctx := context.Background()

	roles, err := roleRepo.GetByMember(ctx, 999999999, 999999999)
	if err != nil {
		t.Fatalf("GetByMember: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("expected empty slice, got %d roles", len(roles))
	}
}
