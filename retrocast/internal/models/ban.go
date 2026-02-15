package models

import "time"

type Ban struct {
	GuildID   int64     `json:"guild_id,string"`
	UserID    int64     `json:"user_id,string"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedBy int64     `json:"created_by,string"`
	CreatedAt time.Time `json:"created_at"`
}
