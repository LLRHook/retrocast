package api

import (
	"bytes"
	"errors"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
)

// Ordinary requests allow 1 MiB. Uploads allow the 10 MiB file maximum
// (SV-FR-051, implemented as 10<<20) plus 64 KiB of multipart framing.
const JSONBodyLimit int64 = 1 << 20
const UploadBodyLimit int64 = (10 << 20) + (64 << 10)

// RequestBodyLimit bounds reads before handlers bind JSON or parse multipart.
// Reading the bounded body here also catches chunked requests and trailing JSON
// bytes before handlers can swallow a bind error or perform a mutation.
func RequestBodyLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		limit := JSONBodyLimit
		if c.Request().Method == http.MethodPost && c.Path() == "/api/v1/channels/:id/attachments" {
			limit = UploadBodyLimit
		}
		req := c.Request()
		if req.ContentLength > limit {
			return echo.ErrStatusRequestEntityTooLarge
		}
		if req.Body == nil {
			return next(c)
		}
		body, err := io.ReadAll(http.MaxBytesReader(c.Response(), req.Body, limit))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				return echo.ErrStatusRequestEntityTooLarge
			}
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		return next(c)
	}
}
