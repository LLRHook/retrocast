package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

func TestDMChannelRepo_Create(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)

	dmID := nextID()
	dm := &models.DMChannel{
		ID:   dmID,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: user1.ID},
			{ID: user2.ID},
		},
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dmID) })

	got, err := repo.GetByID(ctx, dmID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil after Create")
	}
	if got.Type != models.DMTypeDM {
		t.Errorf("Type = %d, want %d", got.Type, models.DMTypeDM)
	}
	if len(got.Recipients) != 2 {
		t.Errorf("Recipients count = %d, want 2", len(got.Recipients))
	}
}

func TestDMChannelRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 999999999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDMChannelRepo_GetByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)

	dmID := nextID()
	dm := &models.DMChannel{
		ID:   dmID,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: user1.ID},
			{ID: user2.ID},
		},
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dmID) })

	channels, err := repo.GetByUserID(ctx, user1.ID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}

	found := false
	for _, ch := range channels {
		if ch.ID == dmID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetByUserID did not return expected DM channel %d", dmID)
	}
}

func TestDMChannelRepo_GetOrCreateDM_Creates(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)

	newID := nextID()
	dm, err := repo.GetOrCreateDM(ctx, user1.ID, user2.ID, newID)
	if err != nil {
		t.Fatalf("GetOrCreateDM: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dm.ID) })

	if dm == nil {
		t.Fatal("GetOrCreateDM returned nil")
	}
	if dm.ID != newID {
		t.Errorf("ID = %d, want %d (newly created)", dm.ID, newID)
	}
	if dm.Type != models.DMTypeDM {
		t.Errorf("Type = %d, want %d", dm.Type, models.DMTypeDM)
	}
}

func TestDMChannelRepo_GetOrCreateDM_ReturnsExisting(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)

	firstID := nextID()
	dm1, err := repo.GetOrCreateDM(ctx, user1.ID, user2.ID, firstID)
	if err != nil {
		t.Fatalf("GetOrCreateDM first: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dm1.ID) })

	secondID := nextID()
	dm2, err := repo.GetOrCreateDM(ctx, user1.ID, user2.ID, secondID)
	if err != nil {
		t.Fatalf("GetOrCreateDM second: %v", err)
	}

	if dm2.ID != dm1.ID {
		t.Cleanup(func() { cleanupDM(t, pool, dm2.ID) })
		t.Errorf("expected existing DM ID %d, got new ID %d", dm1.ID, dm2.ID)
	}
}

func TestDMChannelRepo_AddRecipient(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)
	user3 := createTestUserSimple(t, userRepo)

	dmID := nextID()
	dm := &models.DMChannel{
		ID:   dmID,
		Type: models.DMTypeGroupDM,
		Recipients: []models.User{
			{ID: user1.ID},
			{ID: user2.ID},
		},
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dmID) })

	if err := repo.AddRecipient(ctx, dmID, user3.ID); err != nil {
		t.Fatalf("AddRecipient: %v", err)
	}

	got, err := repo.GetByID(ctx, dmID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Recipients) != 3 {
		t.Errorf("Recipients count = %d, want 3", len(got.Recipients))
	}
}

func TestDMChannelRepo_AddRecipient_Idempotent(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)

	dmID := nextID()
	dm := &models.DMChannel{
		ID:   dmID,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: user1.ID},
			{ID: user2.ID},
		},
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dmID) })

	// Adding an existing recipient should not error (ON CONFLICT DO NOTHING)
	if err := repo.AddRecipient(ctx, dmID, user1.ID); err != nil {
		t.Fatalf("AddRecipient existing: %v", err)
	}
}

func TestDMChannelRepo_IsRecipient(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	repo := NewDMChannelRepository(pool)
	ctx := context.Background()

	user1 := createTestUserSimple(t, userRepo)
	user2 := createTestUserSimple(t, userRepo)
	user3 := createTestUserSimple(t, userRepo)

	dmID := nextID()
	dm := &models.DMChannel{
		ID:   dmID,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: user1.ID},
			{ID: user2.ID},
		},
		CreatedAt: time.Now().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { cleanupDM(t, pool, dmID) })

	is, err := repo.IsRecipient(ctx, dmID, user1.ID)
	if err != nil {
		t.Fatalf("IsRecipient: %v", err)
	}
	if !is {
		t.Error("expected user1 to be a recipient")
	}

	is, err = repo.IsRecipient(ctx, dmID, user3.ID)
	if err != nil {
		t.Fatalf("IsRecipient: %v", err)
	}
	if is {
		t.Error("expected user3 to NOT be a recipient")
	}
}

// cleanupDM deletes a DM channel directly via SQL.
// dm_recipients has ON DELETE CASCADE, so deleting the channel suffices.
func cleanupDM(t *testing.T, pool *pgxpool.Pool, dmID int64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "DELETE FROM dm_channels WHERE id = $1", dmID)
	if err != nil {
		t.Logf("cleanupDM: %v", err)
	}
}
