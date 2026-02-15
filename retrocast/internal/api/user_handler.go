package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// UserHandler handles user profile endpoints.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// GetMe handles GET /api/v1/users/@me.
func (h *UserHandler) GetMe(c echo.Context) error {
	userID := auth.GetUserID(c)

	user, err := h.service.GetByID(c.Request().Context(), userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": user})
}

type updateUserRequest struct {
	DisplayName *string `json:"display_name"`
	Avatar      *string `json:"avatar"`
}

// UpdateMe handles PATCH /api/v1/users/@me.
func (h *UserHandler) UpdateMe(c echo.Context) error {
	userID := auth.GetUserID(c)

	var req updateUserRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	user, err := h.service.UpdateProfile(c.Request().Context(), userID, req.DisplayName, req.Avatar)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": user})
}
