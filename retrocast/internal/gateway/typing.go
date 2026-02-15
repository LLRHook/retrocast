package gateway

import (
	"time"

	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/redis"

	"github.com/labstack/echo/v4"
)

// TypingHandler handles POST /api/v1/channels/:id/typing.
type TypingHandler struct {
	channels database.ChannelRepository
	redis    *redis.Client
	manager  *Manager
}

// NewTypingHandler creates a TypingHandler.
func NewTypingHandler(channels database.ChannelRepository, redisClient *redis.Client, manager *Manager) *TypingHandler {
	return &TypingHandler{
		channels: channels,
		redis:    redisClient,
		manager:  manager,
	}
}

// Handle processes a typing indicator request.
func (h *TypingHandler) Handle(c echo.Context) error {
	channelID, err := parseSnowflake(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid channel id")
	}

	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channel, err := h.channels.GetByID(ctx, channelID)
	if err != nil {
		return echo.NewHTTPError(500, "internal server error")
	}
	if channel == nil {
		return echo.NewHTTPError(404, "channel not found")
	}

	if err := h.redis.SetTyping(ctx, channelID, userID); err != nil {
		return echo.NewHTTPError(500, "internal server error")
	}

	h.manager.DispatchToGuild(channel.GuildID, EventTypingStart, TypingStartData{
		ChannelID: channelID,
		GuildID:   channel.GuildID,
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	})

	return c.NoContent(204)
}
