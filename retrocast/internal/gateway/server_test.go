package gateway

import (
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestForeignOriginRejected(t *testing.T) {
	req := httptest.NewRequest("GET", "https://instance.test/gateway", nil)
	req.Header.Set("Origin", "https://foreign.test")
	if upgrader.CheckOrigin(req) {
		t.Fatal("foreign origin allowed")
	}
}

func TestGatewayOriginPolicy(t *testing.T) {
	m := NewManager(nil, nil, nil, nil)
	m.SetAllowedOrigins("http://localhost:5173")
	e := echo.New()
	e.GET("/gateway", m.HandleWebSocket)
	srv := httptest.NewServer(e)
	defer srv.Close()
	for _, tc := range []struct {
		origin string
		allow  bool
	}{
		{"", true}, {srv.URL, true}, {"http://localhost:5173", true}, {"https" + strings.TrimPrefix(srv.URL, "http"), false}, {"http://foreign.test", false}, {"http://127.0.0.1:1", false},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			h := http.Header{}
			if tc.origin != "" {
				h.Set("Origin", tc.origin)
			}
			conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/gateway", h)
			if tc.allow {
				if err != nil {
					t.Fatal(err)
				}
				_ = conn.Close()
			} else {
				if err == nil {
					_ = conn.Close()
					t.Fatal("foreign upgrade accepted")
				}
				if resp == nil || resp.StatusCode != 403 {
					t.Fatalf("expected 403, got %v", resp)
				}
			}
		})
	}
}
