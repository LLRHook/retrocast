package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type memberRepo struct {
	pool *pgxpool.Pool
}

func NewMemberRepository(pool *pgxpool.Pool) MemberRepository {
	return &memberRepo{pool: pool}
}

func (r *memberRepo) Create(ctx context.Context, member *models.Member) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`INSERT INTO members (guild_id, user_id, nickname, joined_at)
		 VALUES ($1, $2, $3, $4)`,
		member.GuildID, member.UserID, member.Nickname, member.JoinedAt,
	)
	return err
}

func (r *memberRepo) GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Member, error) {
	m := &models.Member{}
	err := queryDB(ctx, r.pool).QueryRow(ctx,
		`SELECT guild_id, user_id, nickname, joined_at
		 FROM members WHERE guild_id = $1 AND user_id = $2`, guildID, userID,
	).Scan(&m.GuildID, &m.UserID, &m.Nickname, &m.JoinedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	roles, err := r.getMemberRoles(ctx, guildID, userID)
	if err != nil {
		return nil, err
	}
	m.Roles = roles
	return m, nil
}

func (r *memberRepo) GetByGuildID(ctx context.Context, guildID int64, limit, offset int) ([]models.Member, error) {
	rows, err := queryDB(ctx, r.pool).Query(ctx,
		`SELECT guild_id, user_id, nickname, joined_at
		 FROM members WHERE guild_id = $1
		 ORDER BY joined_at
		 LIMIT $2 OFFSET $3`, guildID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.GuildID, &m.UserID, &m.Nickname, &m.JoinedAt); err != nil {
			return nil, err
		}
		roles, err := r.getMemberRoles(ctx, m.GuildID, m.UserID)
		if err != nil {
			return nil, err
		}
		m.Roles = roles
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

func (r *memberRepo) Update(ctx context.Context, member *models.Member) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`UPDATE members SET nickname = $3
		 WHERE guild_id = $1 AND user_id = $2`,
		member.GuildID, member.UserID, member.Nickname,
	)
	return err
}

func (r *memberRepo) Delete(ctx context.Context, guildID, userID int64) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`DELETE FROM members WHERE guild_id = $1 AND user_id = $2`, guildID, userID,
	)
	return err
}

func (r *memberRepo) AddRole(ctx context.Context, guildID, userID, roleID int64) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`INSERT INTO member_roles (guild_id, user_id, role_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		guildID, userID, roleID,
	)
	return err
}

func (r *memberRepo) RemoveRole(ctx context.Context, guildID, userID, roleID int64) error {
	_, err := queryDB(ctx, r.pool).Exec(ctx,
		`DELETE FROM member_roles WHERE guild_id = $1 AND user_id = $2 AND role_id = $3`,
		guildID, userID, roleID,
	)
	return err
}

func (r *memberRepo) getMemberRoles(ctx context.Context, guildID, userID int64) ([]int64, error) {
	rows, err := queryDB(ctx, r.pool).Query(ctx,
		`SELECT role_id FROM member_roles WHERE guild_id = $1 AND user_id = $2`,
		guildID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []int64
	for rows.Next() {
		var roleID int64
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roles = append(roles, roleID)
	}
	return roles, rows.Err()
}

func (r *memberRepo) transactionPool() *pgxpool.Pool { return r.pool }

// HydrateUsers performs one identity query for the whole page. Missing users
// retain an absent projection (useful for incomplete imported member records).
func (r *memberRepo) HydrateUsers(ctx context.Context, members []*models.Member) error {
	if len(members) == 0 {
		return nil
	}
	ids := make([]int64, len(members))
	for i, m := range members {
		ids[i] = m.UserID
		m.User = nil
	}
	rows, err := queryDB(ctx, r.pool).Query(ctx, `SELECT id, username, display_name, avatar_hash FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	users := make(map[int64]*models.MemberUser, len(members))
	for rows.Next() {
		u := &models.MemberUser{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarHash); err != nil {
			return err
		}
		users[u.ID] = u
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range members {
		m.User = users[m.UserID]
	}
	return nil
}
