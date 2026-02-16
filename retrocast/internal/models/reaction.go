package models

import "time"

type Reaction struct {
	MessageID int64     `json:"message_id,string"`
	UserID    int64     `json:"user_id,string"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type ReactionCount struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Me    bool   `json:"me"`
}
