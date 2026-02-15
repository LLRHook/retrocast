package models

import "time"

type Invite struct {
	Code      string     `json:"code"`
	GuildID   int64      `json:"guild_id,string"`
	ChannelID *int64     `json:"channel_id,string,omitempty"`
	CreatorID int64      `json:"creator_id,string"`
	MaxUses   int        `json:"max_uses"`
	Uses      int        `json:"uses"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
