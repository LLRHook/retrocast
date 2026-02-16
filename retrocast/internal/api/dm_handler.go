package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// DMHandler handles DM channel endpoints.
type DMHandler struct {
	service *service.DMService
}

// NewDMHandler creates a DMHandler.
func NewDMHandler(svc *service.DMService) *DMHandler {
	return &DMHandler{service: svc}
}

type createDMRequest struct {
	RecipientID  string   `json:"recipient_id"`
	RecipientIDs []string `json:"recipient_ids"`
}

// CreateDM handles POST /users/@me/channels.
// If recipient_ids (array) is provided, creates a group DM.
// Otherwise uses recipient_id for a 1-on-1 DM.
func (h *DMHandler) CreateDM(c echo.Context) error {
	userID := auth.GetUserID(c)

	var req createDMRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Group DM path: recipient_ids takes priority.
	if len(req.RecipientIDs) > 0 {
		dm, err := h.service.CreateGroupDM(c.Request().Context(), userID, req.RecipientIDs)
		if err != nil {
			return mapServiceError(c, err)
		}
		return c.JSON(http.StatusOK, dm)
	}

	// 1-on-1 DM path.
	dm, err := h.service.CreateDM(c.Request().Context(), userID, req.RecipientID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, dm)
}

// ListDMs handles GET /users/@me/channels.
func (h *DMHandler) ListDMs(c echo.Context) error {
	userID := auth.GetUserID(c)

	channels, err := h.service.ListDMs(c.Request().Context(), userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, channels)
}

// AddGroupDMMember handles PUT /channels/:id/recipients/:user_id.
func (h *DMHandler) AddGroupDMMember(c echo.Context) error {
	callerID := auth.GetUserID(c)

	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_CHANNEL", "invalid channel id")
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_USER", "invalid user id")
	}

	if err := h.service.AddGroupDMMember(c.Request().Context(), callerID, channelID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveGroupDMMember handles DELETE /channels/:id/recipients/:user_id.
func (h *DMHandler) RemoveGroupDMMember(c echo.Context) error {
	callerID := auth.GetUserID(c)

	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_CHANNEL", "invalid channel id")
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_USER", "invalid user id")
	}

	if err := h.service.RemoveGroupDMMember(c.Request().Context(), callerID, channelID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
