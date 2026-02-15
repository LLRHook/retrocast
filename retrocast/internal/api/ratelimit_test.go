package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
)

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

func TestRateLimit_FailOpen(t *testing.T) {
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
