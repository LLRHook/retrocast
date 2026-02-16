package models

import "time"

type ReadState struct {
	UserID        int64     `json:"user_id,string"`
	ChannelID     int64     `json:"channel_id,string"`
	LastMessageID int64     `json:"last_message_id,string"`
	MentionCount  int       `json:"mention_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}
