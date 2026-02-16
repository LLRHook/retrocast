package api

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// ReactionHandler handles message reaction endpoints.
type ReactionHandler struct {
	service *service.ReactionService
}

// NewReactionHandler creates a ReactionHandler.
func NewReactionHandler(svc *service.ReactionService) *ReactionHandler {
	return &ReactionHandler{service: svc}
}

// AddReaction handles PUT /api/v1/channels/:id/messages/:message_id/reactions/:emoji/@me.
func (h *ReactionHandler) AddReaction(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	emoji, err := url.PathUnescape(c.Param("emoji"))
	if err != nil || emoji == "" {
		return Error(c, http.StatusBadRequest, "INVALID_EMOJI", "invalid emoji")
	}

	userID := auth.GetUserID(c)

	if err := h.service.AddReaction(c.Request().Context(), channelID, msgID, userID, emoji); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveReaction handles DELETE /api/v1/channels/:id/messages/:message_id/reactions/:emoji/@me.
func (h *ReactionHandler) RemoveReaction(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	emoji, err := url.PathUnescape(c.Param("emoji"))
	if err != nil || emoji == "" {
		return Error(c, http.StatusBadRequest, "INVALID_EMOJI", "invalid emoji")
	}

	userID := auth.GetUserID(c)

	if err := h.service.RemoveReaction(c.Request().Context(), channelID, msgID, userID, emoji); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetReactions handles GET /api/v1/channels/:id/messages/:message_id/reactions/:emoji.
func (h *ReactionHandler) GetReactions(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	emoji, err := url.PathUnescape(c.Param("emoji"))
	if err != nil || emoji == "" {
		return Error(c, http.StatusBadRequest, "INVALID_EMOJI", "invalid emoji")
	}

	userID := auth.GetUserID(c)

	limit := 25
	if l := c.QueryParam("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed < 1 || parsed > 100 {
			return Error(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be 1-100")
		}
		limit = parsed
	}

	userIDs, err := h.service.GetReactions(c.Request().Context(), channelID, msgID, userID, emoji, limit)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, userIDs)
}
