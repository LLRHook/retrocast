package api

import (
	"bytes"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"mime/multipart"

	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestBodyLimit(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, size := range []int64{JSONBodyLimit - 1, JSONBodyLimit, JSONBodyLimit + 1} {
			t.Run(fmt.Sprintf("%d/chunked=%v", size, chunked), func(t *testing.T) {
				e := echo.New()
				e.HTTPErrorHandler = HTTPErrorHandler
				e.Use(RequestBodyLimit)
				e.POST("/json", func(c echo.Context) error {
					_, err := io.Copy(io.Discard, c.Request().Body)
					if err != nil {
						return err
					}
					return c.NoContent(204)
				})
				req := httptest.NewRequest("POST", "/json", strings.NewReader(strings.Repeat(" ", int(size))))
				if chunked {
					req.ContentLength = -1
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				want := 204
				if size > JSONBodyLimit {
					want = 413
				}
				if rec.Code != want {
					t.Fatalf("%d: %s", rec.Code, rec.Body)
				}
				if want == 413 && !strings.Contains(rec.Body.String(), `PAYLOAD_TOO_LARGE`) {
					t.Fatal(rec.Body)
				}
			})
		}
	}
}

func TestUploadBodyLimit(t *testing.T) {
	for _, size := range []int{10 << 20, int(UploadBodyLimit) + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			file, err := writer.CreateFormFile("file", "maximum.txt")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = file.Write(bytes.Repeat([]byte("x"), size)); err != nil {
				t.Fatal(err)
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			e := echo.New()
			e.HTTPErrorHandler = HTTPErrorHandler
			e.Use(RequestBodyLimit)
			e.POST("/api/v1/channels/:id/attachments", func(c echo.Context) error {
				f, err := c.FormFile("file")
				if err != nil {
					return err
				}
				if f.Size != int64(size) {
					t.Fatalf("file size %d", f.Size)
				}
				return c.NoContent(204)
			})
			req := httptest.NewRequest("POST", "/api/v1/channels/1/attachments", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.ContentLength = -1
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			want := 204
			if size > int(UploadBodyLimit) {
				want = 413
			}
			if rec.Code != want {
				t.Fatalf("%d: %s", rec.Code, rec.Body)
			}
		})
	}
}
