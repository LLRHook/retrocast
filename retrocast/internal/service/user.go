package service

import (
	"context"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
)

// UserService handles user profile business logic.
type UserService struct {
	users database.UserRepository
}

// NewUserService creates a UserService.
func NewUserService(users database.UserRepository) *UserService {
	return &UserService{users: users}
}

// GetByID returns the user with the given ID.
func (s *UserService) GetByID(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if user == nil {
		return nil, NotFound("NOT_FOUND", "user not found")
	}
	return user, nil
}

// UpdateProfile updates the authenticated user's display name and/or avatar.
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, displayName *string, avatar *string) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if user == nil {
		return nil, NotFound("NOT_FOUND", "user not found")
	}

	if displayName != nil {
		if len(*displayName) < 1 || len(*displayName) > 32 {
			return nil, BadRequest("INVALID_DISPLAY_NAME", "display name must be 1-32 characters")
		}
		user.DisplayName = *displayName
	}
	if avatar != nil {
		user.AvatarHash = avatar
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	return user, nil
}
