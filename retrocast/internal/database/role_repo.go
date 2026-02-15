package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type roleRepo struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &roleRepo{pool: pool}
}

func (r *roleRepo) Create(ctx context.Context, role *models.Role) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO roles (id, guild_id, name, color, permissions, position, is_default)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		role.ID, role.GuildID, role.Name, role.Color, role.Permissions, role.Position, role.IsDefault,
	)
	return err
}

func (r *roleRepo) GetByID(ctx context.Context, id int64) (*models.Role, error) {
	role := &models.Role{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, guild_id, name, color, permissions, position, is_default
		 FROM roles WHERE id = $1`, id,
	).Scan(&role.ID, &role.GuildID, &role.Name, &role.Color, &role.Permissions, &role.Position, &role.IsDefault)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return role, err
}

func (r *roleRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, guild_id, name, color, permissions, position, is_default
		 FROM roles WHERE guild_id = $1
		 ORDER BY position`, guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.GuildID, &role.Name, &role.Color, &role.Permissions, &role.Position, &role.IsDefault); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *roleRepo) Update(ctx context.Context, role *models.Role) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE roles SET name = $2, color = $3, permissions = $4, position = $5, is_default = $6
		 WHERE id = $1`,
		role.ID, role.Name, role.Color, role.Permissions, role.Position, role.IsDefault,
	)
	return err
}

func (r *roleRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	return err
}

func (r *roleRepo) GetByMember(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.guild_id, r.name, r.color, r.permissions, r.position, r.is_default
		 FROM roles r
		 INNER JOIN member_roles mr ON mr.role_id = r.id
		 WHERE mr.guild_id = $1 AND mr.user_id = $2
		 ORDER BY r.position`, guildID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.GuildID, &role.Name, &role.Color, &role.Permissions, &role.Position, &role.IsDefault); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
