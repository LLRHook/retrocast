package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/service"
)

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error code and message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error sends a JSON error response.
func Error(c echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// errorJSON is an alias for Error (used by some handlers).
var errorJSON = Error

// mapServiceError converts a service-layer error into the appropriate HTTP response.
func mapServiceError(c echo.Context, err error) error {
	var svcErr *service.ServiceError
	if errors.As(err, &svcErr) {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(svcErr.Err, service.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(svcErr.Err, service.ErrForbidden):
			status = http.StatusForbidden
		case errors.Is(svcErr.Err, service.ErrBadRequest):
			status = http.StatusBadRequest
		case errors.Is(svcErr.Err, service.ErrConflict):
			status = http.StatusConflict
		case errors.Is(svcErr.Err, service.ErrUnauthorized):
			status = http.StatusUnauthorized
		case errors.Is(svcErr.Err, service.ErrGone):
			status = http.StatusGone
		case errors.Is(svcErr.Err, service.ErrRoleHierarchy):
			status = http.StatusForbidden
		}
		return Error(c, status, svcErr.Code, svcErr.Message)
	}
	return Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
}
