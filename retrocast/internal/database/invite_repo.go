package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type inviteRepo struct {
	pool *pgxpool.Pool
}

func NewInviteRepository(pool *pgxpool.Pool) InviteRepository {
	return &inviteRepo{pool: pool}
}

func (r *inviteRepo) Create(ctx context.Context, invite *models.Invite) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`INSERT INTO invites (code, guild_id, channel_id, creator_id, max_uses, uses, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		invite.Code, invite.GuildID, invite.ChannelID, invite.CreatorID,
		invite.MaxUses, invite.Uses, invite.ExpiresAt, invite.CreatedAt,
	)
	return err
}

func (r *inviteRepo) GetByCode(ctx context.Context, code string) (*models.Invite, error) {
	inv := &models.Invite{}
	query := `SELECT code, guild_id, channel_id, creator_id, max_uses, uses, expires_at, created_at FROM invites WHERE code = $1`
	args := []any{code}
	if _, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		// Guild deletion locks the parent before cascading to its invites. Acquire
		// the same parent-first order before locking the invite or inserting a member.
		db := queryDB(ctx, r.pool)
		var guildID int64
		if err := db.QueryRow(ctx, "SELECT guild_id FROM invites WHERE code=$1", code).Scan(&guildID); err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		var present int
		if err := db.QueryRow(ctx, "SELECT 1 FROM guilds WHERE id=$1 FOR KEY SHARE", guildID).Scan(&present); err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		// The invite may have been revoked or moved while the parent lock waited.
		query += " AND guild_id=$2 FOR UPDATE"
		args = append(args, guildID)
	}

	err := queryDB(ctx, r.pool).QueryRow(ctx, query, args...).Scan(
		&inv.Code, &inv.GuildID, &inv.ChannelID, &inv.CreatorID,
		&inv.MaxUses, &inv.Uses, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return inv, err
}

func (r *inviteRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Invite, error) {
	rows, err := queryDB(ctx, r.pool).Query(ctx,
		`SELECT code, guild_id, channel_id, creator_id, max_uses, uses, expires_at, created_at
		 FROM invites WHERE guild_id = $1
		 ORDER BY created_at DESC`, guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []models.Invite
	for rows.Next() {
		var inv models.Invite
		if err := rows.Scan(
			&inv.Code, &inv.GuildID, &inv.ChannelID, &inv.CreatorID,
			&inv.MaxUses, &inv.Uses, &inv.ExpiresAt, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

func (r *inviteRepo) IncrementUses(ctx context.Context, code string) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`UPDATE invites SET uses = uses + 1 WHERE code = $1`, code,
	)
	return err
}

func (r *inviteRepo) Delete(ctx context.Context, code string) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx, `DELETE FROM invites WHERE code = $1`, code)
	return err
}

func (r *inviteRepo) transactionPool() *pgxpool.Pool { return r.pool }
