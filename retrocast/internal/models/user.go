package models

import "time"

type User struct {
	ID           int64     `json:"id,string"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	AvatarHash   *string   `json:"avatar_hash,omitempty"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
