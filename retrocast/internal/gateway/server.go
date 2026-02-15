package gateway

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins; tighten in production.
	},
}

// HandleWebSocket handles GET /gateway by upgrading to WebSocket.
func (m *Manager) HandleWebSocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("gateway: upgrade error: %v", err)
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
