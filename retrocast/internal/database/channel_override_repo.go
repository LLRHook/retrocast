package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type channelOverrideRepo struct {
	pool *pgxpool.Pool
}

func NewChannelOverrideRepository(pool *pgxpool.Pool) ChannelOverrideRepository {
	return &channelOverrideRepo{pool: pool}
}

func (r *channelOverrideRepo) Set(ctx context.Context, override *models.ChannelOverride) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO channel_overrides (channel_id, role_id, allow_perms, deny_perms)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (channel_id, role_id)
		 DO UPDATE SET allow_perms = EXCLUDED.allow_perms, deny_perms = EXCLUDED.deny_perms`,
		override.ChannelID, override.RoleID, override.Allow, override.Deny,
	)
	return err
}

func (r *channelOverrideRepo) GetByChannel(ctx context.Context, channelID int64) ([]models.ChannelOverride, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT channel_id, role_id, allow_perms, deny_perms
		 FROM channel_overrides WHERE channel_id = $1`, channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []models.ChannelOverride
	for rows.Next() {
		var o models.ChannelOverride
		if err := rows.Scan(&o.ChannelID, &o.RoleID, &o.Allow, &o.Deny); err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

func (r *channelOverrideRepo) Delete(ctx context.Context, channelID, roleID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM channel_overrides WHERE channel_id = $1 AND role_id = $2`,
		channelID, roleID,
	)
	return err
}
