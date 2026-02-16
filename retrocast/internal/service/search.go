package service

import (
	"context"
	"time"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// SearchService handles message search business logic.
type SearchService struct {
	messages database.MessageRepository
	members  database.MemberRepository
	perms    *PermissionChecker
}

// NewSearchService creates a SearchService.
func NewSearchService(
	messages database.MessageRepository,
	members database.MemberRepository,
	perms *PermissionChecker,
) *SearchService {
	return &SearchService{
		messages: messages,
		members:  members,
		perms:    perms,
	}
}

// SearchMessages searches messages in a guild with full-text search.
func (s *SearchService) SearchMessages(ctx context.Context, guildID, userID int64, query string, authorID *int64, before *time.Time, after *time.Time, limit int) ([]models.MessageWithAuthor, error) {
	if query == "" {
		return nil, BadRequest("INVALID_QUERY", "search query must not be empty")
	}

	// Verify the user is a member of the guild.
	member, err := s.members.GetByGuildAndUser(ctx, guildID, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if member == nil {
		return nil, Forbidden("FORBIDDEN", "you are not a member of this guild")
	}

	// Check that the user has ReadMessageHistory permission at the guild level.
	if err := s.perms.RequireGuildPermissionByPerm(ctx, guildID, userID, permissions.PermReadMessageHistory); err != nil {
		return nil, err
	}

	messages, err := s.messages.SearchMessages(ctx, guildID, query, authorID, before, after, limit)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if messages == nil {
		messages = []models.MessageWithAuthor{}
	}
	return messages, nil
}
