package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type banRepo struct {
	pool *pgxpool.Pool
}

func NewBanRepository(pool *pgxpool.Pool) BanRepository {
	return &banRepo{pool: pool}
}

func (r *banRepo) Create(ctx context.Context, ban *models.Ban) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bans (guild_id, user_id, reason, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		ban.GuildID, ban.UserID, ban.Reason, ban.CreatedBy, ban.CreatedAt,
	)
	return err
}

func (r *banRepo) GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Ban, error) {
	ban := &models.Ban{}
	err := r.pool.QueryRow(ctx,
		`SELECT guild_id, user_id, reason, created_by, created_at
		 FROM bans WHERE guild_id = $1 AND user_id = $2`, guildID, userID,
	).Scan(
		&ban.GuildID, &ban.UserID, &ban.Reason, &ban.CreatedBy, &ban.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return ban, err
}

func (r *banRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Ban, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT guild_id, user_id, reason, created_by, created_at
		 FROM bans WHERE guild_id = $1
		 ORDER BY created_at DESC`, guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []models.Ban
	for rows.Next() {
		var ban models.Ban
		if err := rows.Scan(
			&ban.GuildID, &ban.UserID, &ban.Reason, &ban.CreatedBy, &ban.CreatedAt,
		); err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, rows.Err()
}

func (r *banRepo) Delete(ctx context.Context, guildID, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM bans WHERE guild_id = $1 AND user_id = $2`, guildID, userID)
	return err
}
