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

// Success writes the REST v2 envelope. Empty operations use NoContent instead.
func Success(c echo.Context, status int, data any) error {
	return c.JSON(status, struct {
		Data any `json:"data"`
	}{data})
}

// HTTPErrorHandler normalizes errors raised by Echo and middleware too.
func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status := http.StatusInternalServerError
	message := "internal server error"
	var he *echo.HTTPError
	if errors.As(err, &he) {
		status = he.Code
		if status < 500 {
			if m, ok := he.Message.(string); ok {
				message = m
			}
		}
	}
	code := "INTERNAL"
	switch status {
	case 400:
		code = "BAD_REQUEST"
	case 401:
		code = "UNAUTHORIZED"
	case 403:
		code = "FORBIDDEN"
	case 404:
		code = "NOT_FOUND"
	case 405:
		code = "METHOD_NOT_ALLOWED"
	case 413:
		code = "PAYLOAD_TOO_LARGE"
	case 429:
		code = "RATE_LIMITED"
	}
	if err := Error(c, status, code, message); err != nil {
		c.Logger().Error(err)
	}
}
