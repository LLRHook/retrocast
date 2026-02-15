package models

type Role struct {
	ID          int64  `json:"id,string"`
	GuildID     int64  `json:"guild_id,string"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Permissions int64  `json:"permissions,string"`
	Position    int    `json:"position"`
	IsDefault   bool   `json:"is_default"`
}
