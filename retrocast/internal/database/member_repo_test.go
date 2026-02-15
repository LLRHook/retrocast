package database

import (
	"context"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestMemberRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	member := &models.Member{
		GuildID:  guild.ID,
		UserID:   owner.ID,
		JoinedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, member); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, owner.ID) })

	got, err := repo.GetByGuildAndUser(ctx, guild.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got == nil {
		t.Fatal("GetByGuildAndUser returned nil after Create")
	}
	if got.GuildID != guild.ID {
		t.Errorf("GuildID = %d, want %d", got.GuildID, guild.ID)
	}
	if got.UserID != owner.ID {
		t.Errorf("UserID = %d, want %d", got.UserID, owner.ID)
	}
}

func TestMemberRepo_Create_Duplicate(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	member := &models.Member{
		GuildID:  guild.ID,
		UserID:   owner.ID,
		JoinedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, member); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, owner.ID) })

	err := repo.Create(ctx, member)
	if err == nil {
		t.Fatal("expected error for duplicate member, got nil")
	}
}

func TestMemberRepo_GetByGuildAndUser_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByGuildAndUser(ctx, 999999999, 999999999)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestMemberRepo_GetByGuildID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	for _, uid := range []int64{owner.ID, user2.ID} {
		m := &models.Member{
			GuildID:  guild.ID,
			UserID:   uid,
			JoinedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("Create member %d: %v", uid, err)
		}
		userID := uid
		t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, userID) })
	}

	members, err := repo.GetByGuildID(ctx, guild.ID, 100, 0)
	if err != nil {
		t.Fatalf("GetByGuildID: %v", err)
	}
	if len(members) < 2 {
		t.Fatalf("expected at least 2 members, got %d", len(members))
	}
}

func TestMemberRepo_GetByGuildID_Pagination(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	for _, uid := range []int64{owner.ID, user2.ID} {
		m := &models.Member{
			GuildID:  guild.ID,
			UserID:   uid,
			JoinedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("Create member: %v", err)
		}
		userID := uid
		t.Cleanup(func() { _ = repo.Delete(ctx, guild.ID, userID) })
	}

	// Limit to 1
	members, err := repo.GetByGuildID(ctx, guild.ID, 1, 0)
	if err != nil {
		t.Fatalf("GetByGuildID limit=1: %v", err)
	}
	if len(members) != 1 {
		t.Errorf("expected 1 member with limit=1, got %d", len(members))
	}
}

func TestMemberRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	member := createTestMember(t, repo, guild.ID, owner.ID)

	nick := "New Nickname"
	member.Nickname = &nick
	if err := repo.Update(ctx, member); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByGuildAndUser(ctx, guild.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got.Nickname == nil || *got.Nickname != nick {
		t.Errorf("Nickname = %v, want %q", got.Nickname, nick)
	}
}

func TestMemberRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	repo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)

	_ = createTestMember(t, repo, guild.ID, owner.ID)

	if err := repo.Delete(ctx, guild.ID, owner.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByGuildAndUser(ctx, guild.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}

func TestMemberRepo_AddRole_RemoveRole(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	guildRepo := NewGuildRepository(pool)
	roleRepo := NewRoleRepository(pool)
	memberRepo := NewMemberRepository(pool)
	ctx := context.Background()

	owner := createTestUserSimple(t, userRepo)
	guild := createTestGuild(t, guildRepo, owner.ID)
	_ = createTestMember(t, memberRepo, guild.ID, owner.ID)

	role := &models.Role{
		ID:      nextID(),
		GuildID: guild.ID,
		Name:    "TestRole",
	}
	if err := roleRepo.Create(ctx, role); err != nil {
		t.Fatalf("Create role: %v", err)
	}
	t.Cleanup(func() { _ = roleRepo.Delete(ctx, role.ID) })

	// AddRole
	if err := memberRepo.AddRole(ctx, guild.ID, owner.ID, role.ID); err != nil {
		t.Fatalf("AddRole: %v", err)
	}

	got, err := memberRepo.GetByGuildAndUser(ctx, guild.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	found := false
	for _, rid := range got.Roles {
		if rid == role.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("role not found after AddRole")
	}

	// AddRole idempotent (ON CONFLICT DO NOTHING)
	if err := memberRepo.AddRole(ctx, guild.ID, owner.ID, role.ID); err != nil {
		t.Fatalf("AddRole duplicate: %v", err)
	}

	// RemoveRole
	if err := memberRepo.RemoveRole(ctx, guild.ID, owner.ID, role.ID); err != nil {
		t.Fatalf("RemoveRole: %v", err)
	}

	got, err = memberRepo.GetByGuildAndUser(ctx, guild.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetByGuildAndUser: %v", err)
	}
	for _, rid := range got.Roles {
		if rid == role.ID {
			t.Error("role still present after RemoveRole")
		}
	}
}

// createTestMember inserts a member and registers cleanup.
func createTestMember(t *testing.T, repo MemberRepository, guildID, userID int64) *models.Member {
	t.Helper()
	ctx := context.Background()
	member := &models.Member{
		GuildID:  guildID,
		UserID:   userID,
		JoinedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, member); err != nil {
		t.Fatalf("createTestMember: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, guildID, userID) })
	return member
}
