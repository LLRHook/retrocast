package models

import "time"

type Member struct {
	GuildID  int64    `json:"guild_id,string"`
	UserID   int64    `json:"user_id,string"`
	Nickname *string  `json:"nickname,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
	Roles    []int64  `json:"roles"`
}
