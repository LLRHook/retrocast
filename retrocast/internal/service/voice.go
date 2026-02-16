package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// VoiceService handles voice channel business logic.
type VoiceService struct {
	voiceStates database.VoiceStateRepository
	channels    database.ChannelRepository
	users       database.UserRepository
	gateway     gateway.Dispatcher
	perms       *PermissionChecker
	apiKey      string
	apiSecret   string
}

// NewVoiceService creates a VoiceService.
func NewVoiceService(
	voiceStates database.VoiceStateRepository,
	channels database.ChannelRepository,
	users database.UserRepository,
	gw gateway.Dispatcher,
	perms *PermissionChecker,
	apiKey, apiSecret string,
) *VoiceService {
	return &VoiceService{
		voiceStates: voiceStates,
		channels:    channels,
		users:       users,
		gateway:     gw,
		perms:       perms,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
	}
}

// JoinChannelResponse is returned when a user joins a voice channel.
type JoinChannelResponse struct {
	Token       string              `json:"token"`
	VoiceStates []models.VoiceState `json:"voice_states"`
}

// JoinChannel connects a user to a voice channel.
func (s *VoiceService) JoinChannel(ctx context.Context, channelID, userID int64) (*JoinChannelResponse, error) {
	channel, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if channel == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	if channel.Type != models.ChannelTypeVoice {
		return nil, BadRequest("NOT_VOICE_CHANNEL", "channel is not a voice channel")
	}

	if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermConnect); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if user == nil {
		return nil, NotFound("NOT_FOUND", "user not found")
	}

	roomName := fmt.Sprintf("voice-%d", channelID)
	token, err := s.generateLiveKitToken(roomName, userID, user.Username)
	if err != nil {
		return nil, Internal("INTERNAL", "failed to generate voice token")
	}

	state := &models.VoiceState{
		GuildID:   channel.GuildID,
		ChannelID: channelID,
		UserID:    userID,
		SessionID: roomName,
		SelfMute:  false,
		SelfDeaf:  false,
		JoinedAt:  time.Now(),
	}

	if err := s.voiceStates.Upsert(ctx, state); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	s.gateway.DispatchToGuild(channel.GuildID, gateway.EventVoiceStateUpdate, state)

	states, err := s.voiceStates.GetByChannel(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if states == nil {
		states = []models.VoiceState{}
	}

	return &JoinChannelResponse{
		Token:       token,
		VoiceStates: states,
	}, nil
}

// LeaveChannel disconnects a user from their voice channel in the guild.
func (s *VoiceService) LeaveChannel(ctx context.Context, channelID, userID int64) error {
	channel, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if channel == nil {
		return NotFound("NOT_FOUND", "channel not found")
	}

	existing, err := s.voiceStates.GetByUser(ctx, channel.GuildID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if existing == nil {
		return NotFound("NOT_FOUND", "not in a voice channel")
	}

	if err := s.voiceStates.Delete(ctx, channel.GuildID, userID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	leavePayload := struct {
		GuildID   int64  `json:"guild_id,string"`
		ChannelID *int64 `json:"channel_id"`
		UserID    int64  `json:"user_id,string"`
	}{
		GuildID:   channel.GuildID,
		ChannelID: nil,
		UserID:    userID,
	}

	s.gateway.DispatchToGuild(channel.GuildID, gateway.EventVoiceStateUpdate, leavePayload)

	return nil
}

// GetChannelVoiceStates returns all voice states for a channel.
func (s *VoiceService) GetChannelVoiceStates(ctx context.Context, channelID, userID int64) ([]models.VoiceState, error) {
	channel, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if channel == nil {
		return nil, NotFound("NOT_FOUND", "channel not found")
	}

	if err := s.perms.RequireChannelPermission(ctx, channel.GuildID, channelID, userID, permissions.PermViewChannel); err != nil {
		return nil, err
	}

	states, err := s.voiceStates.GetByChannel(ctx, channelID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if states == nil {
		states = []models.VoiceState{}
	}

	return states, nil
}

// generateLiveKitToken creates a LiveKit-compatible access token using the standard JWT library.
// LiveKit tokens use HS256 with the API secret and include a "video" grant.
func (s *VoiceService) generateLiveKitToken(roomName string, userID int64, username string) (string, error) {
	now := time.Now()
	identity := fmt.Sprintf("%d", userID)

	claims := jwt.MapClaims{
		"iss":   s.apiKey,
		"sub":   identity,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(24 * time.Hour).Unix(),
		"name":  username,
		"video": map[string]interface{}{
			"roomJoin": true,
			"room":     roomName,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.apiSecret))
}
