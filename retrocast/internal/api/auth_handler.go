package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	service *service.AuthService
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         interface{} `json:"user"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	result, err := h.service.Register(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User,
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	result, err := h.service.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	result, err := h.service.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, refreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(c echo.Context) error {
	var req logoutRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	h.service.Logout(c.Request().Context(), req.RefreshToken)

	return c.NoContent(http.StatusNoContent)
}

// TokenService returns the auth token service for middleware use.
// This is exposed through the Dependencies struct in router.go.
var _ = auth.GetUserID // ensure auth import is used
