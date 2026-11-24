package models

import "time"

// MemberUser is the public identity projection; never embed the account model.
type MemberUser struct {
	ID          int64   `json:"id,string"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	AvatarHash  *string `json:"avatar_hash,omitempty"`
}

type Member struct {
	User     *MemberUser `json:"user,omitempty"`
	GuildID  int64       `json:"guild_id,string"`
	UserID   int64       `json:"user_id,string"`
	Nickname *string     `json:"nickname,omitempty"`
	JoinedAt time.Time   `json:"joined_at"`
	Roles    StringIDs   `json:"roles"`
}
