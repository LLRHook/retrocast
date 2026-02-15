package models

import "time"

type Message struct {
	ID        int64      `json:"id,string"`
	ChannelID int64      `json:"channel_id,string"`
	AuthorID  int64      `json:"author_id,string"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`
}

type MessageWithAuthor struct {
	Message
	AuthorUsername    string  `json:"author_username"`
	AuthorDisplayName string  `json:"author_display_name"`
	AuthorAvatarHash  *string `json:"author_avatar_hash,omitempty"`
}
