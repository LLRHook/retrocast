package api

import (
	"context"
	goredis "github.com/redis/go-redis/v9"
	"github.com/victorivanov/retrocast/internal/auth"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/service"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSessionTransportLive(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL required")
	}
	opts, err := goredis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	direct := goredis.NewClient(opts)
	defer func() { _ = direct.Close() }()
	defer direct.Del(context.Background(), "refresh:l28-transport-fixture")
	for _, operation := range []string{"refresh", "logout"} {
		t.Run(operation, func(t *testing.T) {
			rc, err := redisclient.NewClient(url)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rc.Close() }()
			h := NewAuthHandler(service.NewAuthService(nil, auth.NewTokenService("l28-session-fixture"), rc, nil))
			invoke := func() int {
				c, rec := newTestContext(http.MethodPost, "/api/v1/auth/"+operation, strings.NewReader(`{"refresh_token":"l28-transport-fixture"}`))
				if operation == "refresh" {
					err = h.Refresh(c)
				} else {
					err = h.Logout(c)
				}
				if err != nil {
					t.Fatal(err)
				}
				var detail ErrorResponse
				if err := decodeResponse(rec, &detail); err != nil {
					t.Fatal(err)
				}
				expected := "INVALID_TOKEN"
				if rec.Code == 503 {
					expected = "SESSION_UNAVAILABLE"
				}
				if detail.Error.Code != expected {
					t.Fatalf("code %s want %s", detail.Error.Code, expected)
				}
				return rec.Code
			}
			if status := invoke(); status != 401 {
				t.Fatalf("invalid token status %d", status)
			}
			if err := rc.StoreRefreshToken(context.Background(), "l28-transport-fixture", 25, time.Minute); err != nil {
				t.Fatal(err)
			}
			silent := newSilentRedisAt(t, opts.Addr)
			h = NewAuthHandler(service.NewAuthService(nil, auth.NewTokenService("l28-session-fixture"), silent, nil))
			start := time.Now()
			if status := invoke(); status != 503 {
				t.Fatalf("transport status %d, want 503", status)
			}
			if elapsed := time.Since(start); elapsed > 2750*time.Millisecond {
				t.Fatalf("deadline elapsed %s", elapsed)
			}

		})
	}
}
