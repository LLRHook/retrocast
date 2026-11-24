package gateway

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func timedTransport(t *testing.T, timing connectionTiming) (*websocket.Conn, *Connection) {
	t.Helper()
	ready := make(chan *Connection, 1)
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := newConnection(ws, NewManager(nil, nil, nil, nil))
		c.timing = timing
		ready <- c
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); c.readPump() }()
		go func() { defer wg.Done(); c.writePump() }()
		wg.Wait()
		close(done)
	}))
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}
	c := <-ready
	t.Cleanup(func() {
		_ = ws.Close()
		c.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("transport pumps did not stop")
		}
		srv.Close()
	})
	return ws, c
}
func TestTransportHeartbeatsExtendReadDeadline(t *testing.T) {
	ws, c := timedTransport(t, connectionTiming{40 * time.Millisecond, 100 * time.Millisecond, 180 * time.Millisecond})
	errors := make(chan error, 1)
	var acks atomic.Int64
	go func() {
		for {
			var p GatewayPayload
			if err := ws.ReadJSON(&p); err != nil {
				errors <- err
				return
			}
			if p.Op == OpHeartbeat {
				_ = ws.WriteJSON(GatewayPayload{Op: OpHeartbeat})
			}
			if p.Op == OpHeartbeatAck {
				acks.Add(1)
			}
		}
	}()
	timer := time.NewTimer(550 * time.Millisecond)
	defer timer.Stop()
	select {
	case err := <-errors:
		t.Fatalf("heartbeat session closed before 3 read deadlines: %v", err)
	case <-timer.C:
	}
	select {
	case <-c.done:
		t.Fatal("heartbeat session closed")
	default:
	}
	if acks.Load() < 5 {
		t.Fatalf("received only %d heartbeat acknowledgements", acks.Load())
	}
}
func TestTransportSilentSessionStillCloses(t *testing.T) {
	ws, _ := timedTransport(t, connectionTiming{100 * time.Millisecond, 20 * time.Millisecond, 150 * time.Millisecond})
	_ = ws.SetReadDeadline(time.Now().Add(time.Second))
	start := time.Now()
	for {
		var p GatewayPayload
		if err := ws.ReadJSON(&p); err != nil {
			break
		}
	}
	if time.Since(start) > 700*time.Millisecond {
		t.Fatal("silent transport was not closed by server")
	}
}
func TestTransportNonHeartbeatTrafficStillTimesOut(t *testing.T) {
	ws, _ := timedTransport(t, connectionTiming{40 * time.Millisecond, 60 * time.Millisecond, 2 * time.Second})
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if ws.WriteJSON(GatewayPayload{Op: 99}) != nil {
					return
				}
			case <-stopped:
				return
			}
		}
	}()
	_ = ws.SetReadDeadline(time.Now().Add(time.Second))
	start := time.Now()
	for {
		var p GatewayPayload
		if err := ws.ReadJSON(&p); err != nil {
			break
		}
	}
	if time.Since(start) > 700*time.Millisecond {
		t.Fatal("non-heartbeat frames bypassed existing heartbeat timeout")
	}
}
