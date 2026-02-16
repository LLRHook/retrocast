package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type messageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) MessageRepository {
	return &messageRepo{pool: pool}
}

func (r *messageRepo) Create(ctx context.Context, msg *models.Message) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO messages (id, channel_id, author_id, content, created_at, edited_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		msg.ID, msg.ChannelID, msg.AuthorID, msg.Content, msg.CreatedAt, msg.EditedAt,
	)
	return err
}

func (r *messageRepo) GetByID(ctx context.Context, id int64) (*models.MessageWithAuthor, error) {
	m := &models.MessageWithAuthor{}
	err := r.pool.QueryRow(ctx,
		`SELECT m.id, m.channel_id, m.author_id, m.content, m.created_at, m.edited_at,
		        u.username, u.display_name, u.avatar_hash
		 FROM messages m
		 INNER JOIN users u ON u.id = m.author_id
		 WHERE m.id = $1`, id,
	).Scan(
		&m.ID, &m.ChannelID, &m.AuthorID, &m.Content, &m.CreatedAt, &m.EditedAt,
		&m.AuthorUsername, &m.AuthorDisplayName, &m.AuthorAvatarHash,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (r *messageRepo) GetByChannelID(ctx context.Context, channelID int64, before *int64, limit int) ([]models.MessageWithAuthor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.channel_id, m.author_id, m.content, m.created_at, m.edited_at,
		        u.username, u.display_name, u.avatar_hash
		 FROM messages m
		 INNER JOIN users u ON u.id = m.author_id
		 WHERE m.channel_id = $1 AND ($2::BIGINT IS NULL OR m.id < $2)
		 ORDER BY m.id DESC
		 LIMIT $3`,
		channelID, before, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.MessageWithAuthor
	for rows.Next() {
		var m models.MessageWithAuthor
		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.AuthorID, &m.Content, &m.CreatedAt, &m.EditedAt,
			&m.AuthorUsername, &m.AuthorDisplayName, &m.AuthorAvatarHash,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *messageRepo) Update(ctx context.Context, msg *models.Message) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET content = $2, edited_at = $3
		 WHERE id = $1`,
		msg.ID, msg.Content, msg.EditedAt,
	)
	return err
}

func (r *messageRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, id)
	return err
}

func (r *messageRepo) SearchMessages(ctx context.Context, guildID int64, query string, authorID *int64, before *time.Time, after *time.Time, limit int) ([]models.MessageWithAuthor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.channel_id, m.author_id, m.content, m.created_at, m.edited_at,
		        u.username, u.display_name, u.avatar_hash
		 FROM messages m
		 INNER JOIN channels c ON c.id = m.channel_id
		 INNER JOIN users u ON u.id = m.author_id
		 WHERE c.guild_id = $1
		   AND m.search_vector @@ plainto_tsquery('english', $2)
		   AND ($3::BIGINT IS NULL OR m.author_id = $3)
		   AND ($4::TIMESTAMPTZ IS NULL OR m.created_at < $4)
		   AND ($5::TIMESTAMPTZ IS NULL OR m.created_at > $5)
		 ORDER BY ts_rank(m.search_vector, plainto_tsquery('english', $2)) DESC, m.id DESC
		 LIMIT $6`,
		guildID, query, authorID, before, after, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.MessageWithAuthor
	for rows.Next() {
		var m models.MessageWithAuthor
		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.AuthorID, &m.Content, &m.CreatedAt, &m.EditedAt,
			&m.AuthorUsername, &m.AuthorDisplayName, &m.AuthorAvatarHash,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
