package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	users     database.UserRepository
	tokens    *auth.TokenService
	redis     *redis.Client
	snowflake *snowflake.Generator
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(
	users database.UserRepository,
	tokens *auth.TokenService,
	redis *redis.Client,
	sf *snowflake.Generator,
) *AuthHandler {
	return &AuthHandler{
		users:     users,
		tokens:    tokens,
		redis:     redis,
		snowflake: sf,
	}
}

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{2,32}$`)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         models.User `json:"user"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if !usernameRegexp.MatchString(req.Username) {
		return Error(c, http.StatusBadRequest, "INVALID_USERNAME", "username must be 2-32 alphanumeric or underscore characters")
	}
	if len(req.Password) < 6 || len(req.Password) > 128 {
		return Error(c, http.StatusBadRequest, "INVALID_PASSWORD", "password must be 6-128 characters")
	}

	existing, err := h.users.GetByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if existing != nil {
		return Error(c, http.StatusConflict, "USERNAME_TAKEN", "username is already taken")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	user := &models.User{
		ID:           h.snowflake.Generate().Int64(),
		Username:     req.Username,
		DisplayName:  req.Username,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := h.users.Create(c.Request().Context(), user); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return h.issueTokens(c, user)
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

	user, err := h.users.GetByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	if user == nil {
		return Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
	}

	ok, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !ok {
		return Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
	}

	return h.issueTokens(c, user)
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

	if req.RefreshToken == "" {
		return Error(c, http.StatusBadRequest, "MISSING_TOKEN", "refresh_token is required")
	}

	ctx := c.Request().Context()

	userID, err := h.redis.GetRefreshTokenUserID(ctx, req.RefreshToken)
	if err != nil {
		return Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
	}

	// Rotate: delete old token, issue new pair.
	if err := h.redis.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	accessToken, err := h.tokens.GenerateAccessToken(userID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	refreshToken, err := h.tokens.GenerateRefreshToken()
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	if err := h.redis.StoreRefreshToken(ctx, refreshToken, userID, h.tokens.RefreshExpiry()); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusOK, refreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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

	if req.RefreshToken != "" {
		_ = h.redis.DeleteRefreshToken(c.Request().Context(), req.RefreshToken)
	}

	return c.NoContent(http.StatusNoContent)
}

// issueTokens generates access + refresh tokens and returns the auth response.
func (h *AuthHandler) issueTokens(c echo.Context, user *models.User) error {
	accessToken, err := h.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	refreshToken, err := h.tokens.GenerateRefreshToken()
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	ctx := c.Request().Context()
	if err := h.redis.StoreRefreshToken(ctx, refreshToken, user.ID, h.tokens.RefreshExpiry()); err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}

	return c.JSON(http.StatusOK, authResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	})
}
