package models

type ChannelOverride struct {
	ChannelID int64 `json:"channel_id,string"`
	RoleID    int64 `json:"role_id,string"`
	Allow     int64 `json:"allow,string"`
	Deny      int64 `json:"deny,string"`
}
