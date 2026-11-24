package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
)

// MessageHandler handles message CRUD endpoints.
type MessageHandler struct {
	service *service.MessageService
}

// NewMessageHandler creates a MessageHandler.
func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{service: svc}
}

type sendMessageRequest struct {
	Content       string           `json:"content"`
	AttachmentIDs models.StringIDs `json:"attachment_ids"`
}

// SendMessage handles POST /api/v1/channels/:id/messages.
func (h *MessageHandler) SendMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	var req sendMessageRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	full, err := h.service.SendMessage(c.Request().Context(), channelID, userID, req.Content, req.AttachmentIDs...)
	if err != nil {
		return mapServiceError(c, err)
	}

	return Success(c, http.StatusCreated, full)
}

// GetMessages handles GET /api/v1/channels/:id/messages.
func (h *MessageHandler) GetMessages(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed < 1 || parsed > 100 {
			return Error(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be 1-100")
		}
		limit = parsed
	}

	var before *int64
	if b := c.QueryParam("before"); b != "" {
		parsed, err := strconv.ParseInt(b, 10, 64)
		if err != nil {
			return Error(c, http.StatusBadRequest, "INVALID_BEFORE", "invalid before cursor")
		}
		before = &parsed
	}

	messages, err := h.service.GetMessages(c.Request().Context(), channelID, userID, before, limit)
	if err != nil {
		return mapServiceError(c, err)
	}

	return Success(c, http.StatusOK, messages)
}

// GetMessage handles GET /api/v1/channels/:id/messages/:message_id.
func (h *MessageHandler) GetMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)

	msg, err := h.service.GetMessage(c.Request().Context(), channelID, msgID, userID)
	if err != nil {
		return mapServiceError(c, err)
	}

	return Success(c, http.StatusOK, msg)
}

type editMessageRequest struct {
	Content string `json:"content"`
}

// EditMessage handles PATCH /api/v1/channels/:id/messages/:message_id.
func (h *MessageHandler) EditMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)

	var req editMessageRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	full, err := h.service.EditMessage(c.Request().Context(), channelID, msgID, userID, req.Content)
	if err != nil {
		return mapServiceError(c, err)
	}

	return Success(c, http.StatusOK, full)
}

// DeleteMessage handles DELETE /api/v1/channels/:id/messages/:message_id.
func (h *MessageHandler) DeleteMessage(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	msgID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid message ID")
	}

	userID := auth.GetUserID(c)

	if err := h.service.DeleteMessage(c.Request().Context(), channelID, msgID, userID); err != nil {
		return mapServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// Typing handles POST /api/v1/channels/:id/typing.
func (h *MessageHandler) Typing(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return errorJSON(c, http.StatusBadRequest, "INVALID_ID", "invalid channel id")
	}
	if err := h.service.Typing(c.Request().Context(), channelID, auth.GetUserID(c)); err != nil {
		return mapServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
