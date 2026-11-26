package api

import (
	"bytes"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/victorivanov/retrocast/internal/auth"
	"io"
	"mime/multipart"
	"net/http"

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

type bodyReadCounter struct {
	io.Reader
	bytes int64
}

func (r *bodyReadCounter) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.bytes += int64(n)
	return n, err
}

func TestBodyLimitRouterWiring(t *testing.T) {
	oldR, oldG := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldR; prometheus.DefaultGatherer = oldG })
	e := echo.New()
	SetupRouter(e, &Dependencies{Auth: NewAuthHandler(nil), TokenService: auth.NewTokenService("body-limit-router-test"), Redis: newTestRedis(t)})
	t.Run("oversized_json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(strings.Repeat("x", int(JSONBodyLimit)+1)))
		req.ContentLength = -1 // Unknown length must exercise the globally installed lazy cap.
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != 413 || !strings.Contains(rec.Body.String(), "PAYLOAD_TOO_LARGE") {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
		}
	})
	t.Run("unauthenticated_upload_does_not_read", func(t *testing.T) {
		counter := &bodyReadCounter{Reader: strings.NewReader(strings.Repeat("x", int(UploadBodyLimit)))}
		req := httptest.NewRequest("POST", "/api/v1/channels/2000/attachments", counter)
		req.ContentLength = UploadBodyLimit
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != 401 || counter.bytes >= 1<<20 {
			t.Fatalf("status=%d bytes_read=%d", rec.Code, counter.bytes)
		}
	})
	t.Run("metrics_include_413", func(t *testing.T) {
		families, err := registry.Gather()
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range families {
			if f.GetName() != "retrocast_requests_total" {
				continue
			}
			for _, m := range f.Metric {
				for _, l := range m.Label {
					if l.GetValue() == "413" && m.GetCounter().GetValue() >= 1 {
						return
					}
				}
			}
		}
		t.Fatal("413 counter series missing")
	})
}

func TestBodyLimitAfterPublicRateLimit(t *testing.T) {
	oldR, oldG := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldR; prometheus.DefaultGatherer = oldG })
	e := echo.New()
	SetupRouter(e, &Dependencies{Auth: NewAuthHandler(nil), TokenService: auth.NewTokenService("body-limit-router-test"), Redis: newTestRedis(t)})
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("x"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != 400 {
			t.Fatalf("warmup status=%d", rec.Code)
		}
	}
	counter := &bodyReadCounter{Reader: strings.NewReader(strings.Repeat("x", int(JSONBodyLimit)+1))}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", counter)
	req.ContentLength = JSONBodyLimit + 1
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 429 || counter.bytes != 0 {
		t.Fatalf("rate limited: status=%d bytes_read=%d", rec.Code, counter.bytes)
	}
}

func TestBodyLimitHandlerErrorMapping(t *testing.T) {
	for _, kind := range []string{"bind", "multipart"} {
		t.Run(kind, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = HTTPErrorHandler
			e.Use(RequestBodyLimit)
			path, body, ct := "/json", strings.Repeat("x", int(JSONBodyLimit)+1), "application/json"
			if kind == "bind" {
				// A long JSON token forces the binder to encounter MaxBytesError.
				body = `{"username":"` + body + `"}`
				e.POST(path, NewAuthHandler(nil).Login)
			} else {
				path = "/api/v1/channels/1/attachments"
				body = "--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"x.txt\"\r\n\r\n" + strings.Repeat("x", int(UploadBodyLimit)) + "\r\n--x--\r\n"
				ct = "multipart/form-data; boundary=x"
				e.POST("/api/v1/channels/:id/attachments", func(c echo.Context) error { setAuthUser(c, testUserID); return NewUploadHandler(nil).Upload(c) })
			}
			req := httptest.NewRequest("POST", path, strings.NewReader(body))
			req.ContentLength = -1
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != 413 || !strings.Contains(rec.Body.String(), "PAYLOAD_TOO_LARGE") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
			}
		})
	}
}
