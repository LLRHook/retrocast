package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Ordinary requests allow 1 MiB. Uploads allow the 10 MiB file maximum
// (SV-FR-051, implemented as 10<<20) plus 64 KiB of multipart framing.
const JSONBodyLimit int64 = 1 << 20
const UploadBodyLimit int64 = (10 << 20) + (64 << 10)

func requestBodyLimit(c echo.Context) int64 {
	if c.Request().Method == http.MethodPost && c.Path() == "/api/v1/channels/:id/attachments" {
		return UploadBodyLimit
	}
	return JSONBodyLimit
}

type limitedBody struct {
	io.ReadCloser
	exceeded bool
}

func (b *limitedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		b.exceeded = true
	}
	return n, err
}

func bodyLimitExceeded(c echo.Context) bool {
	b, ok := c.Request().Body.(*limitedBody)
	return ok && b.exceeded
}

// RequestBodyLimit only wraps the body. It does not read bytes or reject a
// Content-Length before authentication and rate limiting have run.
func RequestBodyLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		if req.Body != nil {
			req.Body = &limitedBody{ReadCloser: http.MaxBytesReader(c.Response(), req.Body, requestBodyLimit(c))}
		}
		return next(c)
	}
}

// ReadRequestBody runs inside route groups AFTER their auth and rate limit
// middleware. The bounded copy detects trailing JSON bytes and prevents handlers
// (including optional-body binders) from swallowing overflow and mutating state.
func ReadRequestBody(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		if req.ContentLength > requestBodyLimit(c) {
			return echo.ErrStatusRequestEntityTooLarge
		}
		if req.Body == nil {
			return next(c)
		}
		body, err := io.ReadAll(req.Body)
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
