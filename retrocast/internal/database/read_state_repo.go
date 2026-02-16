package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type readStateRepo struct {
	pool *pgxpool.Pool
}

func NewReadStateRepository(pool *pgxpool.Pool) ReadStateRepository {
	return &readStateRepo{pool: pool}
}

func (r *readStateRepo) Upsert(ctx context.Context, userID, channelID, lastMessageID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO read_states (user_id, channel_id, last_message_id, mention_count, updated_at)
		 VALUES ($1, $2, $3, 0, NOW())
		 ON CONFLICT (user_id, channel_id)
		 DO UPDATE SET last_message_id = $3, mention_count = 0, updated_at = NOW()`,
		userID, channelID, lastMessageID,
	)
	return err
}

func (r *readStateRepo) GetByUser(ctx context.Context, userID int64) ([]models.ReadState, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, channel_id, last_message_id, mention_count, updated_at
		 FROM read_states
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []models.ReadState
	for rows.Next() {
		var s models.ReadState
		if err := rows.Scan(&s.UserID, &s.ChannelID, &s.LastMessageID, &s.MentionCount, &s.UpdatedAt); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

func (r *readStateRepo) GetByUserAndChannel(ctx context.Context, userID, channelID int64) (*models.ReadState, error) {
	s := &models.ReadState{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, channel_id, last_message_id, mention_count, updated_at
		 FROM read_states
		 WHERE user_id = $1 AND channel_id = $2`,
		userID, channelID,
	).Scan(&s.UserID, &s.ChannelID, &s.LastMessageID, &s.MentionCount, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *readStateRepo) IncrementMentionCount(ctx context.Context, userID, channelID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE read_states SET mention_count = mention_count + 1, updated_at = NOW()
		 WHERE user_id = $1 AND channel_id = $2`,
		userID, channelID,
	)
	return err
}
