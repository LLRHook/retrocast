package gateway

import (
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	heartbeatInterval = 41250 * time.Millisecond
	heartbeatTimeout  = 10 * time.Second
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	maxMessageSize    = 4096
	sendBufferSize    = 256
)

// Connection represents a single WebSocket client connection.
type Connection struct {
	UserID    int64
	SessionID string
	Conn      *websocket.Conn
	Send      chan []byte
	manager   *Manager
	sequence  atomic.Int64

	closeOnce sync.Once
	done      chan struct{}

	lastHeartbeat atomic.Int64 // unix millis of last heartbeat ACK from client
}

func newConnection(conn *websocket.Conn, manager *Manager) *Connection {
	c := &Connection{
		Conn:    conn,
		Send:    make(chan []byte, sendBufferSize),
		manager: manager,
		done:    make(chan struct{}),
	}
	c.lastHeartbeat.Store(time.Now().UnixMilli())
	return c
}

// NextSequence increments and returns the next sequence number.
func (c *Connection) NextSequence() int64 {
	return c.sequence.Add(1)
}

// SendPayload marshals and queues a payload to be sent.
func (c *Connection) SendPayload(p GatewayPayload) {
	data, err := json.Marshal(p)
	if err != nil {
		slog.Error("marshal error", "userID", c.UserID, "error", err)
		return
	}
	select {
	case c.Send <- data:
	default:
		slog.Warn("send buffer full, dropping message", "userID", c.UserID)
	}
}

// SendEvent sends a dispatch event with a sequence number.
func (c *Connection) SendEvent(name string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		slog.Error("marshal event error", "event", name, "error", err)
		return
	}
	seq := c.NextSequence()
	c.SendPayload(GatewayPayload{
		Op:        OpDispatch,
		Data:      raw,
		Sequence:  &seq,
		Event: &name,
	})
}

// Close terminates the connection.
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.Conn.Close()
	})
}

// readPump reads messages from the WebSocket and handles them.
func (c *Connection) readPump() {
	defer func() {
		c.manager.unregister(c)
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("read error", "userID", c.UserID, "error", err)
			}
			return
		}
		c.handleMessage(message)
	}
}

// writePump writes messages from the Send channel to the WebSocket,
// and sends heartbeats on a timer.
func (c *Connection) writePump() {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer func() {
		heartbeatTicker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-heartbeatTicker.C:
			// Check if client responded to last heartbeat.
			lastAck := c.lastHeartbeat.Load()
			if time.Since(time.UnixMilli(lastAck)) > heartbeatInterval+heartbeatTimeout {
				slog.Warn("heartbeat timeout", "userID", c.UserID)
				return
			}

			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.SendPayload(GatewayPayload{Op: OpHeartbeat})

		case <-c.done:
			return
		}
	}
}

// handleMessage processes an incoming gateway payload from the client.
func (c *Connection) handleMessage(data []byte) {
	var payload GatewayPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		slog.Error("invalid payload", "userID", c.UserID, "error", err)
		return
	}

	switch payload.Op {
	case OpHeartbeat:
		c.lastHeartbeat.Store(time.Now().UnixMilli())
		c.SendPayload(GatewayPayload{Op: OpHeartbeatAck})

	case OpIdentify:
		c.manager.handleIdentify(c, payload.Data)

	case OpResume:
		c.manager.handleResume(c, payload.Data)

	case OpPresenceUpdate:
		c.manager.handlePresenceUpdate(c, payload.Data)
	}
}
