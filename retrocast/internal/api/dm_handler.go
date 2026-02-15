package api

import (
	"net/http"

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
	RecipientID string `json:"recipient_id"`
}

// CreateDM handles POST /users/@me/channels.
func (h *DMHandler) CreateDM(c echo.Context) error {
	userID := auth.GetUserID(c)

	var req createDMRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

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
