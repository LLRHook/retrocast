package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
)

// SetupRouter must install the proxy-aware client IP extractor that main relies on;
// without it every client behind the reverse proxy shares one rate-limit key.
func TestSetupRouter_ResolvesClientIPThroughProxy(t *testing.T) {
	e := echo.New()
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
