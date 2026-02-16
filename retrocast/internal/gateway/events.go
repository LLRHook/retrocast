package gateway

import (
	"encoding/json"

	"github.com/victorivanov/retrocast/internal/models"
)

// Op codes for gateway payloads.
const (
	OpDispatch         = 0
	OpHeartbeat        = 1
	OpIdentify         = 2
	OpPresenceUpdate   = 3
	OpVoiceStateUpdate = 4
	OpResume           = 6
	OpReconnect        = 7
	OpHello            = 10
	OpHeartbeatAck     = 11
)

// Event names for DISPATCH payloads.
const (
	EventReady              = "READY"
	EventMessageCreate      = "MESSAGE_CREATE"
	EventMessageUpdate      = "MESSAGE_UPDATE"
	EventMessageDelete      = "MESSAGE_DELETE"
	EventGuildCreate        = "GUILD_CREATE"
	EventGuildUpdate        = "GUILD_UPDATE"
	EventGuildDelete        = "GUILD_DELETE"
	EventChannelCreate      = "CHANNEL_CREATE"
	EventChannelUpdate      = "CHANNEL_UPDATE"
	EventChannelDelete      = "CHANNEL_DELETE"
	EventGuildMemberAdd     = "GUILD_MEMBER_ADD"
	EventGuildMemberRemove  = "GUILD_MEMBER_REMOVE"
	EventGuildMemberUpdate  = "GUILD_MEMBER_UPDATE"
	EventGuildRoleCreate    = "GUILD_ROLE_CREATE"
	EventGuildRoleUpdate    = "GUILD_ROLE_UPDATE"
	EventGuildRoleDelete    = "GUILD_ROLE_DELETE"
	EventTypingStart        = "TYPING_START"
	EventPresenceUpdate     = "PRESENCE_UPDATE"
	EventVoiceStateUpdate   = "VOICE_STATE_UPDATE"
	EventGuildBanAdd           = "GUILD_BAN_ADD"
	EventGuildBanRemove        = "GUILD_BAN_REMOVE"
	EventMessageReactionAdd    = "MESSAGE_REACTION_ADD"
	EventMessageReactionRemove = "MESSAGE_REACTION_REMOVE"
)

// GatewayPayload is the envelope for all gateway messages.
type GatewayPayload struct {
	Op       int              `json:"op"`
	Data     json.RawMessage  `json:"d,omitempty"`
	Sequence *int64           `json:"s,omitempty"`
	Event    *string          `json:"t,omitempty"`
}

// IdentifyData is sent by the client in an Op 2 IDENTIFY.
type IdentifyData struct {
	Token string `json:"token"`
}

// ResumeData is sent by the client in an Op 6 RESUME.
type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

// HelloData is sent by the server after WebSocket connect.
type HelloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

// ReadyData is sent by the server after successful IDENTIFY.
type ReadyData struct {
	SessionID  string             `json:"session_id"`
	UserID     int64              `json:"user_id,string"`
	Guilds     []int64            `json:"guilds"`
	ReadStates []models.ReadState `json:"read_states"`
}

// Event is a dispatch event ready to broadcast.
type Event struct {
	Name string
	Data any
}

// TypingStartData is the payload for TYPING_START events.
type TypingStartData struct {
	ChannelID int64 `json:"channel_id,string"`
	GuildID   int64 `json:"guild_id,string"`
	UserID    int64 `json:"user_id,string"`
	Timestamp int64 `json:"timestamp"`
}

// PresenceUpdateData is the payload for PRESENCE_UPDATE events.
type PresenceUpdateData struct {
	UserID int64  `json:"user_id,string"`
	Status string `json:"status"`
}

// ClientPresenceUpdate is sent by the client in an Op 3 PRESENCE_UPDATE.
type ClientPresenceUpdate struct {
	Status string `json:"status"`
}
