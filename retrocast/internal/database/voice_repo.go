package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
)

type voiceStateRepo struct {
	pool *pgxpool.Pool
}

func NewVoiceStateRepository(pool *pgxpool.Pool) VoiceStateRepository {
	return &voiceStateRepo{pool: pool}
}

func (r *voiceStateRepo) Upsert(ctx context.Context, state *models.VoiceState) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO voice_states (guild_id, channel_id, user_id, session_id, self_mute, self_deaf, joined_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (guild_id, user_id)
		 DO UPDATE SET channel_id = $2, session_id = $4, self_mute = $5, self_deaf = $6`,
		state.GuildID, state.ChannelID, state.UserID, state.SessionID, state.SelfMute, state.SelfDeaf, state.JoinedAt,
	)
	return err
}

func (r *voiceStateRepo) Delete(ctx context.Context, guildID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM voice_states WHERE guild_id = $1 AND user_id = $2`,
		guildID, userID,
	)
	return err
}

func (r *voiceStateRepo) GetByChannel(ctx context.Context, channelID int64) ([]models.VoiceState, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT guild_id, channel_id, user_id, session_id, self_mute, self_deaf, joined_at
		 FROM voice_states
		 WHERE channel_id = $1
		 ORDER BY joined_at`,
		channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []models.VoiceState
	for rows.Next() {
		var s models.VoiceState
		if err := rows.Scan(&s.GuildID, &s.ChannelID, &s.UserID, &s.SessionID, &s.SelfMute, &s.SelfDeaf, &s.JoinedAt); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

func (r *voiceStateRepo) GetByGuild(ctx context.Context, guildID int64) ([]models.VoiceState, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT guild_id, channel_id, user_id, session_id, self_mute, self_deaf, joined_at
		 FROM voice_states
		 WHERE guild_id = $1
		 ORDER BY joined_at`,
		guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []models.VoiceState
	for rows.Next() {
		var s models.VoiceState
		if err := rows.Scan(&s.GuildID, &s.ChannelID, &s.UserID, &s.SessionID, &s.SelfMute, &s.SelfDeaf, &s.JoinedAt); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

func (r *voiceStateRepo) GetByUser(ctx context.Context, guildID, userID int64) (*models.VoiceState, error) {
	s := &models.VoiceState{}
	err := r.pool.QueryRow(ctx,
		`SELECT guild_id, channel_id, user_id, session_id, self_mute, self_deaf, joined_at
		 FROM voice_states
		 WHERE guild_id = $1 AND user_id = $2`,
		guildID, userID,
	).Scan(&s.GuildID, &s.ChannelID, &s.UserID, &s.SessionID, &s.SelfMute, &s.SelfDeaf, &s.JoinedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return s, err
}
