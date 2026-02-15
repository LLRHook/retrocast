package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type dmChannelRepo struct {
	pool *pgxpool.Pool
}

func NewDMChannelRepository(pool *pgxpool.Pool) DMChannelRepository {
	return &dmChannelRepo{pool: pool}
}

func (r *dmChannelRepo) Create(ctx context.Context, dm *models.DMChannel) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO dm_channels (id, type, created_at) VALUES ($1, $2, $3)`,
		dm.ID, dm.Type, dm.CreatedAt,
	)
	if err != nil {
		return err
	}

	for _, u := range dm.Recipients {
		_, err = tx.Exec(ctx,
			`INSERT INTO dm_recipients (channel_id, user_id) VALUES ($1, $2)`,
			dm.ID, u.ID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *dmChannelRepo) GetByID(ctx context.Context, id int64) (*models.DMChannel, error) {
	dm := &models.DMChannel{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, created_at FROM dm_channels WHERE id = $1`, id,
	).Scan(&dm.ID, &dm.Type, &dm.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	recipients, err := r.getRecipients(ctx, id)
	if err != nil {
		return nil, err
	}
	dm.Recipients = recipients
	return dm, nil
}

func (r *dmChannelRepo) GetByUserID(ctx context.Context, userID int64) ([]models.DMChannel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT dc.id, dc.type, dc.created_at
		 FROM dm_channels dc
		 INNER JOIN dm_recipients dr ON dr.channel_id = dc.id
		 WHERE dr.user_id = $1
		 ORDER BY dc.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.DMChannel
	for rows.Next() {
		var dm models.DMChannel
		if err := rows.Scan(&dm.ID, &dm.Type, &dm.CreatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, dm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range channels {
		recipients, err := r.getRecipients(ctx, channels[i].ID)
		if err != nil {
			return nil, err
		}
		channels[i].Recipients = recipients
	}

	return channels, nil
}

func (r *dmChannelRepo) GetOrCreateDM(ctx context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
	// Find existing DM channel where both users are recipients.
	var existingID int64
	err := r.pool.QueryRow(ctx,
		`SELECT dr1.channel_id
		 FROM dm_recipients dr1
		 INNER JOIN dm_recipients dr2 ON dr1.channel_id = dr2.channel_id
		 INNER JOIN dm_channels dc ON dc.id = dr1.channel_id
		 WHERE dr1.user_id = $1 AND dr2.user_id = $2 AND dc.type = $3`,
		user1ID, user2ID, models.DMTypeDM,
	).Scan(&existingID)

	if err == nil {
		return r.GetByID(ctx, existingID)
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Create new DM channel.
	now := time.Now()
	dm := &models.DMChannel{
		ID:        newID,
		Type:      models.DMTypeDM,
		CreatedAt: now,
		Recipients: []models.User{
			{ID: user1ID},
			{ID: user2ID},
		},
	}

	if err := r.Create(ctx, dm); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, newID)
}

func (r *dmChannelRepo) AddRecipient(ctx context.Context, channelID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO dm_recipients (channel_id, user_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		channelID, userID,
	)
	return err
}

func (r *dmChannelRepo) IsRecipient(ctx context.Context, channelID, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM dm_recipients WHERE channel_id = $1 AND user_id = $2)`,
		channelID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *dmChannelRepo) getRecipients(ctx context.Context, channelID int64) ([]models.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.username, u.display_name, u.avatar_hash, u.created_at
		 FROM users u
		 INNER JOIN dm_recipients dr ON dr.user_id = u.id
		 WHERE dr.channel_id = $1
		 ORDER BY u.id`, channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarHash, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
