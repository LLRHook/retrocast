package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// VoiceHandler handles voice channel endpoints.
type VoiceHandler struct {
	service *service.VoiceService
}

// NewVoiceHandler creates a VoiceHandler.
func NewVoiceHandler(svc *service.VoiceService) *VoiceHandler {
	return &VoiceHandler{service: svc}
}

// JoinVoice handles POST /api/v1/channels/:id/voice/join.
func (h *VoiceHandler) JoinVoice(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	resp, err := h.service.JoinChannel(c.Request().Context(), channelID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// LeaveVoice handles POST /api/v1/channels/:id/voice/leave.
func (h *VoiceHandler) LeaveVoice(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.LeaveChannel(c.Request().Context(), channelID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetVoiceStates handles GET /api/v1/channels/:id/voice/states.
func (h *VoiceHandler) GetVoiceStates(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	states, err := h.service.GetChannelVoiceStates(c.Request().Context(), channelID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, states)
}
