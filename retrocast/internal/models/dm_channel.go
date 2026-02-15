package models

import "time"

type DMChannelType int

const (
	DMTypeDM      DMChannelType = 1
	DMTypeGroupDM DMChannelType = 3
)

type DMChannel struct {
	ID         int64         `json:"id,string"`
	Type       DMChannelType `json:"type"`
	Recipients []User        `json:"recipients"`
	CreatedAt  time.Time     `json:"created_at"`
}
