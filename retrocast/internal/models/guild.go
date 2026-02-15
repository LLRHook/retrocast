package models

import "time"

type Guild struct {
	ID        int64     `json:"id,string"`
	Name      string    `json:"name"`
	IconHash  *string   `json:"icon_hash,omitempty"`
	OwnerID   int64     `json:"owner_id,string"`
	CreatedAt time.Time `json:"created_at"`
}
