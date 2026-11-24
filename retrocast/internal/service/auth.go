package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
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
	registrationMode string
	invites          *InviteService
	users            database.UserRepository
	tokens           *auth.TokenService
	redis            *redis.Client
	snowflake        *snowflake.Generator
}

// NewAuthService creates an AuthService.
func NewAuthService(
	users database.UserRepository,
	tokens *auth.TokenService,
	redis *redis.Client,
	sf *snowflake.Generator,
) *AuthService {
	return &AuthService{
		registrationMode: "invite",
		users:            users,
		tokens:           tokens,
		redis:            redis,
		snowflake:        sf,
	}
}

// ConfigureRegistration sets the validated operator mode and shared invite service.
func (s *AuthService) ConfigureRegistration(mode string, invites *InviteService) {
	s.registrationMode, s.invites = mode, invites
}

// Register creates a new user and returns tokens.
func (s *AuthService) Register(ctx context.Context, username, password string, inviteCodes ...string) (*AuthResult, error) {
	if s.registrationMode == "closed" {
		return nil, Forbidden("REGISTRATION_CLOSED", "registration is closed")
	}
	code := ""
	if len(inviteCodes) > 0 {
		code = inviteCodes[0]
	}

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

	var guild *models.Guild
	var member *models.Member
	err = database.InTransaction(ctx, s.users, func(ctx context.Context) error {
		first, err := database.LockRegistration(ctx, s.users)
		if err != nil {
			return err
		}
		if s.registrationMode == "invite" && code == "" && !first {
			return Forbidden("INVITE_REQUIRED", "an invite code is required")
		}
		if err := s.users.Create(ctx, user); err != nil {
			return err
		}
		if s.registrationMode == "invite" && code != "" {
			if s.invites == nil {
				return Internal("INTERNAL", "invite service unavailable")
			}
			guild, member, err = s.invites.acceptInvite(ctx, code, user.ID)
			if err != nil {
				var se *ServiceError
				if errors.As(err, &se) && !errors.Is(err, ErrInternal) {
					return Forbidden("INVALID_INVITE", "invite is invalid, expired or exhausted")
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		var se *ServiceError
		if errors.As(err, &se) {
			return nil, err
		}
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return nil, Conflict("USERNAME_TAKEN", "username is already taken")
		}
		return nil, Internal("INTERNAL", "internal server error")
	}
	if guild != nil {
		s.invites.joined(user.ID, guild.ID, member)
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

	// Read the epoch before re-reading the hash, so a login verified against an
	// old password cannot mint a token in a newer epoch.
	gen, err := s.redis.RefreshGeneration(ctx, user.ID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	user, err = s.users.GetByID(ctx, user.ID)
	if err != nil || user == nil {
		return nil, Unauthorized("INVALID_CREDENTIALS", "invalid username or password")
	}
	ok, err := auth.VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return nil, Unauthorized("INVALID_CREDENTIALS", "invalid username or password")
	}

	return s.issueTokens(ctx, user, gen)
}

// Refresh rotates a refresh token and returns a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	if refreshToken == "" {
		return nil, BadRequest("MISSING_TOKEN", "refresh_token is required")
	}

	newRefresh, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	userID, err := s.redis.RotateRefreshToken(ctx, refreshToken, newRefresh, s.tokens.RefreshExpiry())
	if err != nil {
		return nil, Unauthorized("INVALID_TOKEN", "invalid or expired refresh token")
	}
	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
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

func (s *AuthService) issueTokens(ctx context.Context, user *models.User, generation ...int64) (*AuthResult, error) {
	accessToken, err := s.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	refreshToken, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	var storeErr error
	if len(generation) > 0 {
		storeErr = s.redis.StoreRefreshTokenAtGeneration(ctx, refreshToken, user.ID, s.tokens.RefreshExpiry(), generation[0])
	} else {
		storeErr = s.redis.StoreRefreshToken(ctx, refreshToken, user.ID, s.tokens.RefreshExpiry())
	}
	if storeErr != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

// ChangePassword revokes all sessions before the database CAS; failures therefore
// leave sessions revoked, even when the password remains unchanged.
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, current, next string) (*AuthResult, error) {
	if len(next) < 6 || len(next) > 128 {
		return nil, BadRequest("INVALID_PASSWORD", "password must be 6-128 characters")
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if user == nil {
		return nil, Forbidden("INVALID_CREDENTIALS", "current password is incorrect")
	}
	ok, err := auth.VerifyPassword(current, user.PasswordHash)
	if err != nil || !ok {
		return nil, Forbidden("INVALID_CREDENTIALS", "current password is incorrect")
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	repo, ok := s.users.(interface {
		CompareAndSwapPassword(context.Context, int64, string, string) (bool, error)
	})
	if !ok {
		return nil, Internal("INTERNAL", "password update unavailable")
	}
	if err := s.redis.AdvanceRefreshGeneration(ctx, userID); err != nil {
		return nil, Internal("INTERNAL", "could not revoke sessions")
	}
	changed, err := repo.CompareAndSwapPassword(ctx, userID, user.PasswordHash, hash)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if !changed {
		return nil, Forbidden("INVALID_CREDENTIALS", "current password is incorrect")
	}
	return s.issueTokens(ctx, user)
}
