package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type channelRepo struct {
	pool *pgxpool.Pool
}

func NewChannelRepository(pool *pgxpool.Pool) ChannelRepository {
	return &channelRepo{pool: pool}
}

func (r *channelRepo) Create(ctx context.Context, ch *models.Channel) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO channels (id, guild_id, name, type, position, topic, parent_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		ch.ID, ch.GuildID, ch.Name, ch.Type, ch.Position, ch.Topic, ch.ParentID,
	)
	return err
}

func (r *channelRepo) GetByID(ctx context.Context, id int64) (*models.Channel, error) {
	ch := &models.Channel{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, guild_id, name, type, position, topic, parent_id
		 FROM channels WHERE id = $1`, id,
	).Scan(&ch.ID, &ch.GuildID, &ch.Name, &ch.Type, &ch.Position, &ch.Topic, &ch.ParentID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return ch, err
}

func (r *channelRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Channel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, guild_id, name, type, position, topic, parent_id
		 FROM channels WHERE guild_id = $1
		 ORDER BY position, id`, guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var ch models.Channel
		if err := rows.Scan(&ch.ID, &ch.GuildID, &ch.Name, &ch.Type, &ch.Position, &ch.Topic, &ch.ParentID); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *channelRepo) Update(ctx context.Context, ch *models.Channel) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels SET name = $2, type = $3, position = $4, topic = $5, parent_id = $6
		 WHERE id = $1`,
		ch.ID, ch.Name, ch.Type, ch.Position, ch.Topic, ch.ParentID,
	)
	return err
}

func (r *channelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels WHERE id = $1`, id)
	return err
}
