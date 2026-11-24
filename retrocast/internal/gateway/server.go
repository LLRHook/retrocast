package gateway

import (
	"github.com/victorivanov/retrocast/internal/origin"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return (origin.Policy{}).Check(r)
	},
}

// SetAllowedOrigins configures browser origins before serving requests.
func (m *Manager) SetAllowedOrigins(allowed string) { m.allowedOrigins = allowed }

// HandleWebSocket handles GET /gateway by upgrading to WebSocket.
func (m *Manager) HandleWebSocket(c echo.Context) error {
	u := upgrader
	u.CheckOrigin = (origin.Policy{Allowed: m.allowedOrigins}).Check
	ws, err := u.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("websocket upgrade error", "error", err)
		return nil
	}

	conn := newConnection(ws, m)

	// Send HELLO with heartbeat interval.
	conn.SendPayload(GatewayPayload{
		Op: OpHello,
		Data: mustMarshal(HelloData{
			HeartbeatInterval: int(heartbeatInterval.Milliseconds()),
		}),
	})

	go conn.writePump()
	go conn.readPump()

	return nil
}
