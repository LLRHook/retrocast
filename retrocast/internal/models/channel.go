package models

type ChannelType int

const (
	ChannelTypeText     ChannelType = 0
	ChannelTypeVoice    ChannelType = 2
	ChannelTypeCategory ChannelType = 4
)

type Channel struct {
	ID       int64       `json:"id,string"`
	GuildID  int64       `json:"guild_id,string"`
	Name     string      `json:"name"`
	Type     ChannelType `json:"type"`
	Position int         `json:"position"`
	Topic    *string     `json:"topic,omitempty"`
	ParentID *int64      `json:"parent_id,string,omitempty"`
}
