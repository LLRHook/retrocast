package models

type MemberRole struct {
	GuildID int64 `json:"guild_id,string"`
	UserID  int64 `json:"user_id,string"`
	RoleID  int64 `json:"role_id,string"`
}
