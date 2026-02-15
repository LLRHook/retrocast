package service

import (
	"context"
	"regexp"
	"time"

	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{2,32}$`)

// AuthResult holds the tokens and user returned after registration or login.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         models.User
}

// RefreshResult holds the new token pair after a refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

// AuthService handles registration, login, token refresh, and logout.
type AuthService struct {
	users     database.UserRepository
	tokens    *auth.TokenService
	redis     *redis.Client
	snowflake *snowflake.Generator
}

// NewAuthService creates an AuthService.
func NewAuthService(
	users database.UserRepository,
	tokens *auth.TokenService,
	redis *redis.Client,
	sf *snowflake.Generator,
) *AuthService {
	return &AuthService{
		users:     users,
		tokens:    tokens,
		redis:     redis,
		snowflake: sf,
	}
}

// Register creates a new user and returns tokens.
func (s *AuthService) Register(ctx context.Context, username, password string) (*AuthResult, error) {
	if !usernameRegexp.MatchString(username) {
		return nil, BadRequest("INVALID_USERNAME", "username must be 2-32 alphanumeric or underscore characters")
	}
	if len(password) < 6 || len(password) > 128 {
		return nil, BadRequest("INVALID_PASSWORD", "password must be 6-128 characters")
	}

	existing, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if existing != nil {
		return nil, Conflict("USERNAME_TAKEN", "username is already taken")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	user := &models.User{
		ID:           s.snowflake.Generate().Int64(),
		Username:     username,
		DisplayName:  username,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return s.issueTokens(ctx, user)
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthResult, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if user == nil {
		return nil, Unauthorized("INVALID_CREDENTIALS", "invalid username or password")
	}

	ok, err := auth.VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return nil, Unauthorized("INVALID_CREDENTIALS", "invalid username or password")
	}

	return s.issueTokens(ctx, user)
}

// Refresh rotates a refresh token and returns a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	if refreshToken == "" {
		return nil, BadRequest("MISSING_TOKEN", "refresh_token is required")
	}

	userID, err := s.redis.GetRefreshTokenUserID(ctx, refreshToken)
	if err != nil {
		return nil, Unauthorized("INVALID_TOKEN", "invalid or expired refresh token")
	}

	if err := s.redis.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	newRefresh, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if err := s.redis.StoreRefreshToken(ctx, newRefresh, userID, s.tokens.RefreshExpiry()); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}

// Logout deletes the given refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) {
	if refreshToken != "" {
		_ = s.redis.DeleteRefreshToken(ctx, refreshToken)
	}
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (*AuthResult, error) {
	accessToken, err := s.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	refreshToken, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if err := s.redis.StoreRefreshToken(ctx, refreshToken, user.ID, s.tokens.RefreshExpiry()); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}
