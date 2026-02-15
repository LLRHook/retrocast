package api

import "github.com/labstack/echo/v4"

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

// successJSON sends a JSON success response with a data envelope.
func successJSON(c echo.Context, status int, data interface{}) error {
	return c.JSON(status, map[string]interface{}{"data": data})
}
