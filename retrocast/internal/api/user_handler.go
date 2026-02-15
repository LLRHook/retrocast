package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
)

// UserHandler handles user profile endpoints.
type UserHandler struct {
	users database.UserRepository
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(users database.UserRepository) *UserHandler {
	return &UserHandler{users: users}
}

// GetMe handles GET /api/v1/users/@me.
func (h *UserHandler) GetMe(c echo.Context) error {
	userID := auth.GetUserID(c)

	user, err := h.users.GetByID(c.Request().Context(), userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if user == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "user not found")
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

	user, err := h.users.GetByID(c.Request().Context(), userID)
	if err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if user == nil {
		return errorJSON(c, http.StatusNotFound, "NOT_FOUND", "user not found")
	}

	var req updateUserRequest
	if err := c.Bind(&req); err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if req.DisplayName != nil {
		if len(*req.DisplayName) < 1 || len(*req.DisplayName) > 32 {
			return errorJSON(c, http.StatusBadRequest, "INVALID_DISPLAY_NAME", "display name must be 1-32 characters")
		}
		user.DisplayName = *req.DisplayName
	}
	if req.Avatar != nil {
		user.AvatarHash = req.Avatar
	}

	if err := h.users.Update(c.Request().Context(), user); err != nil {
		return errorJSON(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusOK, map[string]any{"data": user})
}
