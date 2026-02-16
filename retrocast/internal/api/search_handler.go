package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// SearchHandler handles message search endpoints.
type SearchHandler struct {
	service *service.SearchService
}

// NewSearchHandler creates a SearchHandler.
func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{service: svc}
}

// SearchMessages handles GET /api/v1/guilds/:id/messages/search.
func (h *SearchHandler) SearchMessages(c echo.Context) error {
	guildID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid guild ID")
	}

	userID := auth.GetUserID(c)

	query := c.QueryParam("q")
	if query == "" {
		return Error(c, http.StatusBadRequest, "INVALID_QUERY", "search query is required")
	}

	limit := 25
	if l := c.QueryParam("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed < 1 || parsed > 100 {
			return Error(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be 1-100")
		}
		limit = parsed
	}

	var authorID *int64
	if a := c.QueryParam("author_id"); a != "" {
		parsed, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return Error(c, http.StatusBadRequest, "INVALID_AUTHOR_ID", "invalid author_id")
		}
		authorID = &parsed
	}

	var before *time.Time
	if b := c.QueryParam("before"); b != "" {
		parsed, err := time.Parse(time.RFC3339, b)
		if err != nil {
			return Error(c, http.StatusBadRequest, "INVALID_BEFORE", "before must be RFC3339 format")
		}
		before = &parsed
	}

	var after *time.Time
	if a := c.QueryParam("after"); a != "" {
		parsed, err := time.Parse(time.RFC3339, a)
		if err != nil {
			return Error(c, http.StatusBadRequest, "INVALID_AFTER", "after must be RFC3339 format")
		}
		after = &parsed
	}

	messages, err := h.service.SearchMessages(c.Request().Context(), guildID, userID, query, authorID, before, after, limit)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, messages)
}
