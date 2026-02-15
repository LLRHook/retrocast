package database

import (
	"context"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/models"
)

func TestUserRepo_Create(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &models.User{
		ID:           nextID(),
		Username:     "testuser_create",
		DisplayName:  "Test User",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, user.ID) })

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID after Create: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil after Create")
	}
	if got.Username != user.Username {
		t.Errorf("Username = %q, want %q", got.Username, user.Username)
	}
	if got.DisplayName != user.DisplayName {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, user.DisplayName)
	}
}

func TestUserRepo_Create_DuplicateUsername(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user1 := &models.User{
		ID:           nextID(),
		Username:     "testuser_dup",
		DisplayName:  "Test User 1",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	user2 := &models.User{
		ID:           nextID(),
		Username:     "testuser_dup",
		DisplayName:  "Test User 2",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}

	err := repo.Create(ctx, user1)
	if err != nil {
		t.Fatalf("Create user1: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, user1.ID) })

	err = repo.Create(ctx, user2)
	if err == nil {
		t.Cleanup(func() { _ = repo.Delete(ctx, user2.ID) })
		t.Fatal("expected error for duplicate username, got nil")
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent ID, got %+v", got)
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &models.User{
		ID:           nextID(),
		Username:     "testuser_getbyname",
		DisplayName:  "By Name",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, user.ID) })

	got, err := repo.GetByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got == nil {
		t.Fatal("GetByUsername returned nil")
	}
	if got.ID != user.ID {
		t.Errorf("ID = %d, want %d", got.ID, user.ID)
	}
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByUsername(ctx, "nonexistent_user_xyz")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent username, got %+v", got)
	}
}

func TestUserRepo_Update(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &models.User{
		ID:           nextID(),
		Username:     "testuser_update",
		DisplayName:  "Before Update",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, user.ID) })

	user.DisplayName = "After Update"
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.DisplayName != "After Update" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "After Update")
	}
}

func TestUserRepo_Delete(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &models.User{
		ID:           nextID(),
		Username:     "testuser_delete",
		DisplayName:  "To Delete",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abc$def",
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID after Delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Delete")
	}
}
