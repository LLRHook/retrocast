package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/service"
)

// UploadHandler handles file upload endpoints.
type UploadHandler struct {
	service *service.UploadService
}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler(svc *service.UploadService) *UploadHandler {
	return &UploadHandler{service: svc}
}

// Upload handles POST /api/v1/channels/:id/attachments.
func (h *UploadHandler) Upload(c echo.Context) error {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Error(c, http.StatusBadRequest, "INVALID_ID", "invalid channel ID")
	}

	userID := auth.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		return Error(c, http.StatusBadRequest, "MISSING_FILE", "file field is required")
	}

	src, err := file.Open()
	if err != nil {
		return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")

	attachment, err := h.service.UploadFile(c.Request().Context(), channelID, userID, file.Filename, file.Size, contentType, src)
	if err != nil {
		return mapServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, attachment)
}
