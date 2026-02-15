package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, username, display_name, avatar_hash, password_hash, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Username, user.DisplayName, user.AvatarHash, user.PasswordHash, user.CreatedAt,
	)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, display_name, avatar_hash, password_hash, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarHash, &u.PasswordHash, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, display_name, avatar_hash, password_hash, created_at
		 FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarHash, &u.PasswordHash, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET username = $2, display_name = $3, avatar_hash = $4, password_hash = $5
		 WHERE id = $1`,
		user.ID, user.Username, user.DisplayName, user.AvatarHash, user.PasswordHash,
	)
	return err
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
