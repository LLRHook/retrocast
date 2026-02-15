package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type attachmentRepo struct {
	pool *pgxpool.Pool
}

func NewAttachmentRepository(pool *pgxpool.Pool) AttachmentRepository {
	return &attachmentRepo{pool: pool}
}

func (r *attachmentRepo) Create(ctx context.Context, a *models.Attachment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO attachments (id, message_id, filename, content_type, size, storage_key)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.MessageID, a.Filename, a.ContentType, a.Size, a.StorageKey,
	)
	return err
}

func (r *attachmentRepo) GetByMessageID(ctx context.Context, messageID int64) ([]models.Attachment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, message_id, filename, content_type, size, storage_key
		 FROM attachments
		 WHERE message_id = $1
		 ORDER BY id`, messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []models.Attachment
	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.Filename, &a.ContentType, &a.Size, &a.StorageKey); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

func (r *attachmentRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	return err
}
