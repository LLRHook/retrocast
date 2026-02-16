package models

import "time"

type VoiceState struct {
	GuildID   int64     `json:"guild_id,string"`
	ChannelID int64     `json:"channel_id,string"`
	UserID    int64     `json:"user_id,string"`
	SessionID string    `json:"session_id"`
	SelfMute  bool      `json:"self_mute"`
	SelfDeaf  bool      `json:"self_deaf"`
	JoinedAt  time.Time `json:"joined_at"`
}
