package gateway

import (
	"context"
	"time"

	"github.com/victorivanov/retrocast/internal/redis"
)

// PresenceService provides presence query operations backed by Redis.
type PresenceService struct {
	redis *redis.Client
}

// NewPresenceService creates a PresenceService.
func NewPresenceService(redisClient *redis.Client) *PresenceService {
	return &PresenceService{redis: redisClient}
}

// GetStatus returns the current status for a user (online, idle, dnd, or empty for offline).
func (ps *PresenceService) GetStatus(userID int64) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, err := ps.redis.GetPresence(ctx, userID)
	if err != nil || status == "" {
		return "offline"
	}
	return status
}

// SetOnline marks a user as online.
func (ps *PresenceService) SetOnline(userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return ps.redis.SetPresence(ctx, userID, "online")
}

// SetOffline marks a user as offline.
func (ps *PresenceService) SetOffline(userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return ps.redis.DeletePresence(ctx, userID)
}
