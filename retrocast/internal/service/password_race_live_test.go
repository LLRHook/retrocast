package service

import (
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/snowflake"
	"os"
	"testing"
	"time"
)

// readGate returns the real MVCC snapshot only after the password change commits.
// It proves that a login must retain the epoch read before this snapshot.
type passwordReadGate struct {
	database.UserRepository
	read    chan struct{}
	release chan struct{}
}

func (r *passwordReadGate) GetByID(ctx context.Context, id int64) (*models.User, error) {
	user, err := r.UserRepository.GetByID(ctx, id)
	close(r.read)
	select {
	case <-r.release:
		return user, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestPasswordLoginRaceLive(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("REDIS_URL") == "" {
		t.Skip("DATABASE_URL and REDIS_URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	rdb, err := redisclient.NewClient(os.Getenv("REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rdb.Close() }()
	users := database.NewUserRepository(pool)
	sf, _ := snowflake.NewGenerator(8, 8)
	svc := NewAuthService(users, auth.NewTokenService("local-password-race-fixture"), rdb, sf)
	svc.ConfigureRegistration("open", nil)
	for _, scenario := range []string{"login during CAS wait", "login store after final advance"} {
		t.Run(scenario, func(t *testing.T) {
			name := fmt.Sprintf("l25race%d", sf.Generate().Int64())
			account, err := svc.Register(ctx, name, "password123")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = users.Delete(context.Background(), account.User.ID) }()
			if scenario == "login during CAS wait" {
				// A separate connection owns a real row lock. The CAS cannot commit until released.
				tx, err := pool.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = tx.Rollback(context.Background()) }()
				if _, err = tx.Exec(ctx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", account.User.ID); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() { _, e := svc.ChangePassword(ctx, account.User.ID, "password123", "changed123"); done <- e }()
				ticker := time.NewTicker(5 * time.Millisecond)
				defer ticker.Stop()
				for {
					gen, e := rdb.RefreshGeneration(ctx, account.User.ID)
					if e != nil {
						t.Fatal(e)
					}
					if gen > 0 {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("change did not advance before CAS")
					case <-ticker.C:
					}
				}
				login, e := svc.Login(ctx, name, "password123")
				if e != nil {
					t.Fatal("old-password fixture login during CAS must succeed")
				}
				if err = tx.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				if err = <-done; err != nil {
					t.Fatal(err)
				}
				if _, err = svc.Refresh(ctx, login.RefreshToken); err == nil {
					t.Fatal("old-password login during CAS survived the completed password change")
				}
			} else {
				gate := &passwordReadGate{UserRepository: users, read: make(chan struct{}), release: make(chan struct{})}
				gated := NewAuthService(gate, svc.tokens, rdb, sf)
				type result struct {
					value *AuthResult
					err   error
				}
				done := make(chan result, 1)
				go func() { v, e := gated.Login(ctx, name, "password123"); done <- result{v, e} }()
				select {
				case <-gate.read:
				case <-ctx.Done():
					t.Fatal("login did not read hash")
				}
				_, err = svc.ChangePassword(ctx, account.User.ID, "password123", "changed123")
				close(gate.release)
				if err != nil {
					t.Fatal(err)
				}
				login := <-done
				if login.err == nil {
					if _, e := svc.Refresh(ctx, login.value.RefreshToken); e == nil {
						t.Fatal("login without pinned generation survived")
					}
				}
			}
			if _, err = svc.Login(ctx, name, "password123"); err == nil {
				t.Fatal("old password accepted after change")
			}
			pair, err := svc.Login(ctx, name, "changed123")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Refresh(ctx, pair.RefreshToken); err != nil {
				t.Fatal("new password session refused")
			}
		})
	}
}

type passwordDeadlineRepo struct {
	database.UserRepository
	user     *models.User
	casCalls int
}

func (r *passwordDeadlineRepo) GetByID(context.Context, int64) (*models.User, error) {
	return r.user, nil
}
func (r *passwordDeadlineRepo) CompareAndSwapPassword(context.Context, int64, string, string) (bool, error) {
	r.casCalls++
	return true, nil
}
func TestPasswordRedisDeadline(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rdb.Close() }()
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	repo := &passwordDeadlineRepo{user: &models.User{ID: 25, PasswordHash: hash}}
	svc := NewAuthService(repo, auth.NewTokenService("deadline-fixture"), rdb, nil)
	mr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	start := time.Now()
	_, err = svc.ChangePassword(ctx, 25, "password123", "changed123")
	if err == nil || repo.casCalls != 0 {
		t.Fatal("unavailable Redis must refuse password change before CAS")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("password Redis deadline exceeded: %s", elapsed)
	}
}
