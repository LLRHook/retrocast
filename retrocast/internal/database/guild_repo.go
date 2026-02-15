package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type guildRepo struct {
	pool *pgxpool.Pool
}

func NewGuildRepository(pool *pgxpool.Pool) GuildRepository {
	return &guildRepo{pool: pool}
}

func (r *guildRepo) Create(ctx context.Context, guild *models.Guild) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO guilds (id, name, icon_hash, owner_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		guild.ID, guild.Name, guild.IconHash, guild.OwnerID, guild.CreatedAt,
	)
	return err
}

func (r *guildRepo) GetByID(ctx context.Context, id int64) (*models.Guild, error) {
	g := &models.Guild{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, icon_hash, owner_id, created_at
		 FROM guilds WHERE id = $1`, id,
	).Scan(&g.ID, &g.Name, &g.IconHash, &g.OwnerID, &g.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return g, err
}

func (r *guildRepo) Update(ctx context.Context, guild *models.Guild) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE guilds SET name = $2, icon_hash = $3, owner_id = $4
		 WHERE id = $1`,
		guild.ID, guild.Name, guild.IconHash, guild.OwnerID,
	)
	return err
}

func (r *guildRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM guilds WHERE id = $1`, id)
	return err
}

func (r *guildRepo) GetByUserID(ctx context.Context, userID int64) ([]models.Guild, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.name, g.icon_hash, g.owner_id, g.created_at
		 FROM guilds g
		 INNER JOIN members m ON m.guild_id = g.id
		 WHERE m.user_id = $1
		 ORDER BY g.id`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guilds []models.Guild
	for rows.Next() {
		var g models.Guild
		if err := rows.Scan(&g.ID, &g.Name, &g.IconHash, &g.OwnerID, &g.CreatedAt); err != nil {
			return nil, err
		}
		guilds = append(guilds, g)
	}
	return guilds, rows.Err()
}
