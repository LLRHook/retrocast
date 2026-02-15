package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// DMHandler handles DM channel endpoints.
type DMHandler struct {
	dms     database.DMChannelRepository
	users   database.UserRepository
	sf      *snowflake.Generator
	gateway gateway.Dispatcher
}

// NewDMHandler creates a DMHandler.
func NewDMHandler(
	dms database.DMChannelRepository,
	users database.UserRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
) *DMHandler {
	return &DMHandler{
		dms:     dms,
		users:   users,
		sf:      sf,
		gateway: gw,
	}
}

type createDMRequest struct {
	RecipientID string `json:"recipient_id"`
}

// CreateDM handles POST /users/@me/channels.
func (h *DMHandler) CreateDM(c echo.Context) error {
	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	var req createDMRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	recipientID, err := strconv.ParseInt(req.RecipientID, 10, 64)
	if err != nil || recipientID == 0 {
		return Error(c, http.StatusBadRequest, "INVALID_RECIPIENT", "invalid recipient_id")
	}

	if recipientID == userID {
		return Error(c, http.StatusBadRequest, "INVALID_RECIPIENT", "cannot create DM with yourself")
	}

	recipient, err := h.users.GetByID(ctx, recipientID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if recipient == nil {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "recipient not found")
	}

	newID := h.sf.Generate().Int64()
	dm, err := h.dms.GetOrCreateDM(ctx, userID, recipientID, newID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	// If the channel was just created (its ID matches newID), dispatch CHANNEL_CREATE.
	if dm.ID == newID {
		h.gateway.DispatchToUser(userID, gateway.EventChannelCreate, dm)
		h.gateway.DispatchToUser(recipientID, gateway.EventChannelCreate, dm)
	}

	return c.JSON(http.StatusOK, dm)
}

// ListDMs handles GET /users/@me/channels.
func (h *DMHandler) ListDMs(c echo.Context) error {
	userID := auth.GetUserID(c)
	ctx := c.Request().Context()

	channels, err := h.dms.GetByUserID(ctx, userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if channels == nil {
		channels = []models.DMChannel{}
	}

	return c.JSON(http.StatusOK, channels)
}
