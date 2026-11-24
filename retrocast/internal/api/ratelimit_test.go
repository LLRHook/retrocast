package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
)

func TestRateLimit_ForwardedHeadersCannotRotateKey(t *testing.T) {
	rdb := newTestRedis(t)
	wrapped := RateLimitMiddleware(rdb, 1, time.Minute)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	for i, tc := range []struct {
		peer      string
		forwarded string
		want      int
	}{
		{"192.0.2.1:1234", "198.51.100.1", http.StatusOK},
		{"192.0.2.1:5678", "198.51.100.2", http.StatusTooManyRequests},
		{"192.0.2.2:1234", "198.51.100.2", http.StatusOK},
	} {
		c, rec := newTestContext(http.MethodGet, "/api/v1/test", nil)
		c.Request().RemoteAddr = tc.peer
		c.Request().Header.Set("X-Forwarded-For", tc.forwarded)
		c.Request().Header.Set("X-Real-IP", tc.forwarded)
		if err := wrapped(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tc.want {
			t.Fatalf("request %d: got %d, want %d", i, rec.Code, tc.want)
		}
	}
}

func TestRateLimit_ClientIPExtractorSeparatesProxiedClients(t *testing.T) {
	rdb := newTestRedis(t)
	e := echo.New()
	e.IPExtractor = ClientIPExtractor()
	e.GET("/api/v1/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, RateLimitMiddleware(rdb, 1, time.Minute))

	const proxy = "172.18.0.5:41000" // Caddy on the compose network (private address)
	for i, tc := range []struct {
		peer      string
		forwarded string
		want      int
	}{
		// Two clients behind the trusted proxy are limited separately.
		{proxy, "203.0.113.10", http.StatusOK},
		{proxy, "203.0.113.10", http.StatusTooManyRequests},
		{proxy, "203.0.113.11", http.StatusOK},
		// A client connecting directly from a public address cannot rotate its key.
		{"198.51.100.7:5000", "203.0.113.20", http.StatusOK},
		{"198.51.100.7:5001", "203.0.113.21", http.StatusTooManyRequests},
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.RemoteAddr = tc.peer
		req.Header.Set(echo.HeaderXForwardedFor, tc.forwarded)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("request %d (peer %s, X-Forwarded-For %s): got %d, want %d", i, tc.peer, tc.forwarded, rec.Code, tc.want)
		}
	}
}

func TestRateLimit_Allowed(t *testing.T) {
	rdb := newTestRedis(t)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	mw := RateLimitMiddleware(rdb, 5, time.Minute)
	wrapped := mw(handler)

	c, rec := newTestContext(http.MethodGet, "/api/v1/test", nil)
	setAuthUser(c, 1)

	if err := wrapped(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify rate limit headers are present.
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Errorf("expected X-RateLimit-Limit=5, got %q", got)
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "4" {
		t.Errorf("expected X-RateLimit-Remaining=4, got %q", got)
	}
	if got := rec.Header().Get("X-RateLimit-Reset"); got == "" {
		t.Error("expected X-RateLimit-Reset header to be set")
	}
}

func TestRateLimit_Exceeded(t *testing.T) {
	rdb := newTestRedis(t)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := RateLimitMiddleware(rdb, 2, time.Minute)
	wrapped := mw(handler)

	// Use up the limit.
	for i := 0; i < 2; i++ {
		c, _ := newTestContext(http.MethodGet, "/api/v1/test", nil)
		if err := wrapped(c); err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
	}

	// Third request should be rate limited.
	c, rec := newTestContext(http.MethodGet, "/api/v1/test", nil)
	if err := wrapped(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d: %s", http.StatusTooManyRequests, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "RATE_LIMITED" {
		t.Errorf("expected error code 'RATE_LIMITED', got %q", errResp.Error.Code)
	}

	// Verify rate limit headers on 429 response.
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Errorf("expected X-RateLimit-Limit=2, got %q", got)
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("expected X-RateLimit-Remaining=0, got %q", got)
	}
	if got := rec.Header().Get("Retry-After"); got == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

func TestRateLimit_AuthenticatedFailOpen(t *testing.T) {
	// Start miniredis, then close it immediately to simulate Redis failure.
	mr := miniredis.RunT(t)
	rdb, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatalf("creating test redis client: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	mr.Close()

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	mw := RateLimitMiddleware(rdb, 1, time.Minute)
	wrapped := mw(handler)

	c, rec := newTestContext(http.MethodGet, "/api/v1/test", nil)
	setAuthUser(c, 1)

	if err := wrapped(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called on Redis failure (fail-open)")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRateLimit_AuthenticatedUser(t *testing.T) {
	rdb := newTestRedis(t)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := RateLimitMiddleware(rdb, 1, time.Minute)
	wrapped := mw(handler)

	// First request as user 1 — should pass.
	c1, rec1 := newTestContext(http.MethodGet, "/api/v1/test", nil)
	setAuthUser(c1, 1)
	if err := wrapped(c1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Second request as user 1 — should be rate limited.
	c2, rec2 := newTestContext(http.MethodGet, "/api/v1/test", nil)
	setAuthUser(c2, 1)
	if err := wrapped(c2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, rec2.Code)
	}

	// Request as user 2 — different key, should pass.
	c3, rec3 := newTestContext(http.MethodGet, "/api/v1/test", nil)
	setAuthUser(c3, 2)
	if err := wrapped(c3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec3.Code)
	}
}

func TestRateLimit_UnauthenticatedFailClosed(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rdb.Close() }()
	mr.Close()
	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/login", nil)
	err = RateLimitMiddleware(rdb, 5, time.Minute)(func(c echo.Context) error { t.Error("handler called during outage"); return c.NoContent(200) })(c)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != 503 {
		t.Fatalf("got %d, want 503", rec.Code)
	}
}
