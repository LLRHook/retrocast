package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// ReadStateHandler handles read state endpoints.
type ReadStateHandler struct {
	service *service.ReadStateService
}

// NewReadStateHandler creates a ReadStateHandler.
func NewReadStateHandler(svc *service.ReadStateService) *ReadStateHandler {
	return &ReadStateHandler{service: svc}
}

// Ack handles PUT /api/v1/channels/:id/ack/:message_id.
func (h *ReadStateHandler) Ack(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	messageID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.Ack(c.Request().Context(), channelID, messageID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetReadStates handles GET /api/v1/users/@me/read-states.
func (h *ReadStateHandler) GetReadStates(c echo.Context) error {
	userID := auth.GetUserID(c)

	states, err := h.service.GetReadStates(c.Request().Context(), userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, states)
}
