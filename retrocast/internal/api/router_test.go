package api

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/victorivanov/retrocast/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
)

// SetupRouter must install the proxy-aware client IP extractor that main relies on;
// without it every client behind the reverse proxy shares one rate-limit key.
func TestSetupRouter_ResolvesClientIPThroughProxy(t *testing.T) {
	e := echo.New()
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	SetupRouter(e, &Dependencies{TokenService: auth.NewTokenService("router-test")})
	if e.IPExtractor == nil {
		t.Fatal("SetupRouter did not set an IP extractor")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "172.18.0.5:41000" // reverse proxy on a private network
	req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.10")
	if got := e.IPExtractor(req); got != "203.0.113.10" {
		t.Fatalf("client behind proxy: got %q, want %q", got, "203.0.113.10")
	}

	req.RemoteAddr = "198.51.100.7:5000" // direct client on a public address
	if got := e.IPExtractor(req); got != "198.51.100.7" {
		t.Fatalf("direct public client: got %q, want %q", got, "198.51.100.7")
	}
}

func TestInviteLookupLimit(t *testing.T) {
	e := echo.New()
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	SetupRouter(e, &Dependencies{TokenService: auth.NewTokenService("test"), Redis: newTestRedis(t), Invites: newInviteHandler(&mockInviteRepo{GetByCodeFn: func(context.Context, string) (*models.Invite, error) { return nil, nil }}, &mockGuildRepo{}, &mockMemberRepo{}, &mockRoleRepo{}, &mockBanRepo{}, &mockGateway{})})
	for i := 0; i < 6; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/invites/test", nil)
		e.ServeHTTP(rec, req)
		want := 404
		if i == 5 {
			want = 429
		}
		if rec.Code != want {
			t.Fatalf("request %d: got %d want %d", i+1, rec.Code, want)
		}
	}
}
func TestDocsOffline(t *testing.T) {
	e := echo.New()
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	SetupRouter(e, &Dependencies{TokenService: auth.NewTokenService("test")})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("GET", "/docs", nil))
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "unpkg.com") || !strings.Contains(rec.Body.String(), "openapi:") {
		t.Fatalf("docs must serve embedded raw OpenAPI")
	}
}

func TestRouterOriginPolicy(t *testing.T) {
	for _, tc := range []struct {
		origin string
		allow  bool
	}{
		{"", true}, {"https://instance.test", true}, {"https://instance.test:443", true},
		{"http://instance.test", false}, {"https://instance.test:444", false}, {"https://foreign.test", false},
		{"http://localhost:5173", true}, {"null", false},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			e := echo.New()
			prometheus.DefaultRegisterer = prometheus.NewRegistry()
			SetupRouter(e, &Dependencies{TokenService: auth.NewTokenService("test"), AllowedOrigins: "http://localhost:5173"})
			for _, method := range []string{"GET", "OPTIONS"} {
				req := httptest.NewRequest(method, "http://instance.test/docs", nil)
				req.Header.Set("Origin", tc.origin)
				req.Header.Set("X-Forwarded-Proto", "https")
				if method == "OPTIONS" {
					req.Header.Set("Access-Control-Request-Method", "GET")
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				got := rec.Header().Get("Access-Control-Allow-Origin")
				if got == "*" || (tc.origin != "" && (got == tc.origin) != tc.allow) {
					t.Fatalf("%s origin %q: ACAO %q", method, tc.origin, got)
				}
				if tc.origin == "" && method == "GET" && rec.Code != 200 {
					t.Fatalf("native request: %d", rec.Code)
				}
			}
		})
	}
}

func TestPasswordCredentialLimit(t *testing.T) {
	e := echo.New()
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	tokens := auth.NewTokenService("password-limit-fixture")
	rdb := newTestRedis(t)
	SetupRouter(e, &Dependencies{TokenService: tokens, Redis: rdb, Auth: newTestAuthHandler(t, &mockUserRepo{})})
	token, err := tokens.GenerateAccessToken(25)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/@me/password", strings.NewReader(`{"current_password":"wrong","new_password":"tiny"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if i < 5 {
			if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_PASSWORD") {
				t.Fatalf("attempt %d status %d", i+1, rec.Code)
			}
		} else {
			if rec.Code != 429 || !strings.Contains(rec.Body.String(), "RATE_LIMITED") || rec.Header().Get("Retry-After") == "" {
				t.Fatal("sixth credential attempt must be rate limited with Retry-After")
			}
		}
	}
}
