package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type reactionRepo struct {
	pool *pgxpool.Pool
}

func NewReactionRepository(pool *pgxpool.Pool) ReactionRepository {
	return &reactionRepo{pool: pool}
}

func (r *reactionRepo) Add(ctx context.Context, messageID, userID int64, emoji string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO reactions (message_id, user_id, emoji)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (message_id, user_id, emoji) DO NOTHING`,
		messageID, userID, emoji,
	)
	return err
}

func (r *reactionRepo) Remove(ctx context.Context, messageID, userID int64, emoji string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`,
		messageID, userID, emoji,
	)
	return err
}

func (r *reactionRepo) GetByMessage(ctx context.Context, messageID int64) ([]models.Reaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT message_id, user_id, emoji, created_at
		 FROM reactions
		 WHERE message_id = $1
		 ORDER BY created_at`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []models.Reaction
	for rows.Next() {
		var reaction models.Reaction
		if err := rows.Scan(&reaction.MessageID, &reaction.UserID, &reaction.Emoji, &reaction.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, reaction)
	}
	return reactions, rows.Err()
}

func (r *reactionRepo) GetCountsByMessage(ctx context.Context, messageID, currentUserID int64) ([]models.ReactionCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT emoji,
		        COUNT(*) AS count,
		        BOOL_OR(user_id = $2) AS me
		 FROM reactions
		 WHERE message_id = $1
		 GROUP BY emoji
		 ORDER BY MIN(created_at)`,
		messageID, currentUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []models.ReactionCount
	for rows.Next() {
		var rc models.ReactionCount
		if err := rows.Scan(&rc.Emoji, &rc.Count, &rc.Me); err != nil {
			return nil, err
		}
		counts = append(counts, rc)
	}
	return counts, rows.Err()
}

func (r *reactionRepo) GetUsersByReaction(ctx context.Context, messageID int64, emoji string, limit int) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id
		 FROM reactions
		 WHERE message_id = $1 AND emoji = $2
		 ORDER BY created_at
		 LIMIT $3`,
		messageID, emoji, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, rows.Err()
}
