package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestRedis(t *testing.T) *redisclient.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatalf("creating test redis client: %v", err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func newTestManager(t *testing.T, guilds *mockGuildRepo) *Manager {
	t.Helper()
	tokens := auth.NewTokenService("test-secret")
	rdb := newTestRedis(t)
	return NewManager(tokens, guilds, rdb)
}

// fakeConn creates a Connection wired into the Manager with a buffered Send
// channel so we can read dispatched events without a real WebSocket.
// It uses a minimal websocket.Conn that is never written to; the Send channel
// is what we inspect.
func fakeConn(m *Manager, userID int64, sessionID string) *Connection {
	// We need a real *websocket.Conn to avoid nil panics in SendPayload.
	// Use a throw-away test server pair; we won't actually read/write the ws.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Keep the server-side connection alive until test cleanup.
		// Just block until the connection is closed.
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		// If dial fails, create a connection with the server-side ws anyway.
		// This should not happen in practice.
		panic("fakeConn: dial failed: " + err.Error())
	}

	c := &Connection{
		UserID:    userID,
		SessionID: sessionID,
		Conn:      ws,
		Send:      make(chan []byte, sendBufferSize),
		manager:   m,
		done:      make(chan struct{}),
	}
	c.lastHeartbeat.Store(time.Now().UnixMilli())

	// Register the connection in the manager.
	m.mu.Lock()
	m.connections[userID] = c
	m.sessions[sessionID] = c
	m.mu.Unlock()

	return c
}

// drainEvents reads all buffered payloads from a connection's Send channel
// and returns them as decoded GatewayPayload slices.
func drainEvents(c *Connection) []GatewayPayload {
	var payloads []GatewayPayload
	for {
		select {
		case raw := <-c.Send:
			var p GatewayPayload
			if err := json.Unmarshal(raw, &p); err == nil {
				payloads = append(payloads, p)
			}
		default:
			return payloads
		}
	}
}

// mockGuildRepo implements database.GuildRepository for testing.
type mockGuildRepo struct {
	GetByUserIDFn func(ctx context.Context, userID int64) ([]models.Guild, error)
}

func (m *mockGuildRepo) Create(context.Context, *models.Guild) error          { return nil }
func (m *mockGuildRepo) GetByID(context.Context, int64) (*models.Guild, error) { return nil, nil }
func (m *mockGuildRepo) Update(context.Context, *models.Guild) error          { return nil }
func (m *mockGuildRepo) Delete(context.Context, int64) error                  { return nil }
func (m *mockGuildRepo) GetByUserID(ctx context.Context, userID int64) ([]models.Guild, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Ring Buffer Tests
// ---------------------------------------------------------------------------

func TestRingBuffer_AddAndSinceZero(t *testing.T) {
	rb := newRingBuffer(100)
	rb.add(Event{Name: "A", Data: "one"})
	rb.add(Event{Name: "B", Data: "two"})

	events := rb.since(0)
	if len(events) != 2 {
		t.Fatalf("since(0) returned %d events, want 2", len(events))
	}
	if events[0].Name != "A" {
		t.Errorf("events[0].Name = %q, want %q", events[0].Name, "A")
	}
	if events[1].Name != "B" {
		t.Errorf("events[1].Name = %q, want %q", events[1].Name, "B")
	}
}

func TestRingBuffer_SequenceIncrement(t *testing.T) {
	rb := newRingBuffer(100)
	rb.add(Event{Name: "A"})
	rb.add(Event{Name: "B"})
	rb.add(Event{Name: "C"})

	if rb.seq != 3 {
		t.Fatalf("seq = %d, want 3", rb.seq)
	}

	// since(1) should return events with seq > 1, i.e. B(2) and C(3).
	events := rb.since(1)
	if len(events) != 2 {
		t.Fatalf("since(1) returned %d events, want 2", len(events))
	}
	if events[0].Name != "B" {
		t.Errorf("events[0].Name = %q, want %q", events[0].Name, "B")
	}
	if events[1].Name != "C" {
		t.Errorf("events[1].Name = %q, want %q", events[1].Name, "C")
	}
}

func TestRingBuffer_SinceMidway(t *testing.T) {
	rb := newRingBuffer(100)
	for i := 0; i < 10; i++ {
		rb.add(Event{Name: "E"})
	}

	// Since seq 7 should return events 8, 9, 10.
	events := rb.since(7)
	if len(events) != 3 {
		t.Fatalf("since(7) returned %d events, want 3", len(events))
	}
}

func TestRingBuffer_SinceAll(t *testing.T) {
	rb := newRingBuffer(100)
	for i := 0; i < 5; i++ {
		rb.add(Event{Name: "E"})
	}

	// since(5) means "after seq 5" — the last event is seq 5, so nothing returned.
	events := rb.since(5)
	if len(events) != 0 {
		t.Fatalf("since(5) returned %d events, want 0", len(events))
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := newRingBuffer(10)

	// Write 25 events into a buffer of size 10.
	for i := 1; i <= 25; i++ {
		rb.add(Event{Name: "E", Data: i})
	}

	// Buffer should be full and contain the last 10 events (seq 16-25).
	events := rb.since(0)
	if len(events) != 10 {
		t.Fatalf("since(0) after wrap returned %d events, want 10", len(events))
	}

	// The oldest event should have data=16 (seq 16).
	if events[0].Data != 16 {
		t.Errorf("oldest event data = %v, want 16", events[0].Data)
	}
	// The newest event should have data=25 (seq 25).
	if events[9].Data != 25 {
		t.Errorf("newest event data = %v, want 25", events[9].Data)
	}
}

func TestRingBuffer_WrapSincePartial(t *testing.T) {
	rb := newRingBuffer(10)

	for i := 1; i <= 25; i++ {
		rb.add(Event{Name: "E", Data: i})
	}

	// since(20) should return events with seq 21-25 → 5 events.
	events := rb.since(20)
	if len(events) != 5 {
		t.Fatalf("since(20) returned %d events, want 5", len(events))
	}
	if events[0].Data != 21 {
		t.Errorf("events[0].Data = %v, want 21", events[0].Data)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := newRingBuffer(100)
	events := rb.since(0)
	if len(events) != 0 {
		t.Fatalf("since(0) on empty buffer returned %d events, want 0", len(events))
	}
}

func TestRingBuffer_ExactlyFull(t *testing.T) {
	rb := newRingBuffer(5)

	for i := 1; i <= 5; i++ {
		rb.add(Event{Name: "E", Data: i})
	}

	events := rb.since(0)
	if len(events) != 5 {
		t.Fatalf("since(0) returned %d events, want 5", len(events))
	}
	if events[0].Data != 1 {
		t.Errorf("events[0].Data = %v, want 1", events[0].Data)
	}
	if events[4].Data != 5 {
		t.Errorf("events[4].Data = %v, want 5", events[4].Data)
	}
}

// ---------------------------------------------------------------------------
// Subscription Tests
// ---------------------------------------------------------------------------

func TestSubscribe_AddsUserToGuild(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	m.SubscribeToGuild(100, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	members, ok := m.subscriptions[1]
	if !ok {
		t.Fatal("guild 1 not in subscriptions")
	}
	if !members[100] {
		t.Error("user 100 not subscribed to guild 1")
	}
}

func TestSubscribe_MultipleUsersToSameGuild(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	m.SubscribeToGuild(300, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	members := m.subscriptions[1]
	if len(members) != 3 {
		t.Fatalf("guild 1 has %d members, want 3", len(members))
	}
	for _, uid := range []int64{100, 200, 300} {
		if !members[uid] {
			t.Errorf("user %d not subscribed to guild 1", uid)
		}
	}
}

func TestUnsubscribe_RemovesUser(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	m.UnsubscribeFromGuild(100, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	members := m.subscriptions[1]
	if members[100] {
		t.Error("user 100 should not be subscribed after unsubscribe")
	}
	if !members[200] {
		t.Error("user 200 should still be subscribed")
	}
}

func TestUnsubscribe_CleansUpEmptyGuild(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	m.SubscribeToGuild(100, 1)
	m.UnsubscribeFromGuild(100, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.subscriptions[1]; ok {
		t.Error("guild 1 should be removed from subscriptions when empty")
	}
}

func TestUnsubscribe_NonSubscribedUserIsNoop(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	// Unsubscribe from a guild the user was never in.
	m.UnsubscribeFromGuild(999, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.subscriptions[1]; ok {
		t.Error("guild 1 should not exist in subscriptions")
	}
}

func TestSubscribe_UserToMultipleGuilds(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(100, 2)
	m.SubscribeToGuild(100, 3)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, gid := range []int64{1, 2, 3} {
		if !m.subscriptions[gid][100] {
			t.Errorf("user 100 not subscribed to guild %d", gid)
		}
	}
}

// ---------------------------------------------------------------------------
// Dispatch Tests
// ---------------------------------------------------------------------------

func TestDispatchToGuild_SendsToAllSubscribed(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	c2 := fakeConn(m, 200, "s2")
	c3 := fakeConn(m, 300, "s3")
	defer c1.Conn.Close()
	defer c2.Conn.Close()
	defer c3.Conn.Close()

	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	// User 300 is NOT subscribed to guild 1.

	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"content": "hello"})

	// Allow time for the Send channel to be populated.
	time.Sleep(10 * time.Millisecond)

	p1 := drainEvents(c1)
	p2 := drainEvents(c2)
	p3 := drainEvents(c3)

	if len(p1) != 1 {
		t.Errorf("user 100 received %d events, want 1", len(p1))
	}
	if len(p2) != 1 {
		t.Errorf("user 200 received %d events, want 1", len(p2))
	}
	if len(p3) != 0 {
		t.Errorf("user 300 (not subscribed) received %d events, want 0", len(p3))
	}

	// Verify the event name is correct.
	if p1[0].Event == nil || *p1[0].Event != EventMessageCreate {
		t.Errorf("event name = %v, want %q", p1[0].Event, EventMessageCreate)
	}
}

func TestDispatchToGuild_StoresInReplayBuffer(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	defer c1.Conn.Close()
	m.SubscribeToGuild(100, 1)

	m.DispatchToGuild(1, EventMessageCreate, "msg1")
	m.DispatchToGuild(1, EventMessageCreate, "msg2")

	m.replayMu.RLock()
	rb, ok := m.replayBuffer[1]
	m.replayMu.RUnlock()

	if !ok {
		t.Fatal("replay buffer not created for guild 1")
	}

	events := rb.since(0)
	if len(events) != 2 {
		t.Fatalf("replay buffer has %d events, want 2", len(events))
	}
}

func TestDispatchToUser_SendsOnlyToTarget(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	c2 := fakeConn(m, 200, "s2")
	defer c1.Conn.Close()
	defer c2.Conn.Close()

	m.DispatchToUser(100, EventReady, map[string]string{"hello": "world"})

	time.Sleep(10 * time.Millisecond)

	p1 := drainEvents(c1)
	p2 := drainEvents(c2)

	if len(p1) != 1 {
		t.Errorf("target user received %d events, want 1", len(p1))
	}
	if len(p2) != 0 {
		t.Errorf("non-target user received %d events, want 0", len(p2))
	}
}

func TestDispatchToUser_NonExistentUserIsNoop(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	// Should not panic.
	m.DispatchToUser(999, EventReady, "data")
}

func TestDispatchToGuildExcept_ExcludesSpecifiedUser(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	c2 := fakeConn(m, 200, "s2")
	c3 := fakeConn(m, 300, "s3")
	defer c1.Conn.Close()
	defer c2.Conn.Close()
	defer c3.Conn.Close()

	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	m.SubscribeToGuild(300, 1)

	m.DispatchToGuildExcept(1, 200, EventMessageCreate, "hello")

	time.Sleep(10 * time.Millisecond)

	p1 := drainEvents(c1)
	p2 := drainEvents(c2)
	p3 := drainEvents(c3)

	if len(p1) != 1 {
		t.Errorf("user 100 received %d events, want 1", len(p1))
	}
	if len(p2) != 0 {
		t.Errorf("user 200 (excluded) received %d events, want 0", len(p2))
	}
	if len(p3) != 1 {
		t.Errorf("user 300 received %d events, want 1", len(p3))
	}
}

func TestDispatchToGuildExcept_StoresInReplayBuffer(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	defer c1.Conn.Close()
	m.SubscribeToGuild(100, 1)

	m.DispatchToGuildExcept(1, 100, EventMessageCreate, "msg")

	m.replayMu.RLock()
	rb, ok := m.replayBuffer[1]
	m.replayMu.RUnlock()

	if !ok {
		t.Fatal("replay buffer not created for guild 1")
	}

	events := rb.since(0)
	if len(events) != 1 {
		t.Fatalf("replay buffer has %d events, want 1", len(events))
	}
}

func TestDispatchToGuild_NonExistentGuildIsNoop(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	// Should not panic.
	m.DispatchToGuild(999, EventMessageCreate, "data")
}

func TestDispatchToGuildExcept_NonExistentGuildIsNoop(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	// Should not panic.
	m.DispatchToGuildExcept(999, 100, EventMessageCreate, "data")
}

// ---------------------------------------------------------------------------
// Register / Unregister Tests
// ---------------------------------------------------------------------------

func TestRegister_DisplacesExistingConnection(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	defer c1.Conn.Close()

	// Register a second connection for the same user.
	c2 := &Connection{
		UserID:    100,
		SessionID: "s2",
		Conn:      c1.Conn, // reuse for simplicity
		Send:      make(chan []byte, sendBufferSize),
		manager:   m,
		done:      make(chan struct{}),
	}
	c2.lastHeartbeat.Store(time.Now().UnixMilli())

	m.register(c2)

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.connections[100] != c2 {
		t.Error("new connection should replace old one")
	}
	if _, ok := m.sessions["s1"]; ok {
		t.Error("old session should be removed")
	}
	if m.sessions["s2"] != c2 {
		t.Error("new session should be registered")
	}
}

func TestUnregister_RemovesFromAllGuildSubscriptions(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c := fakeConn(m, 100, "s1")
	defer c.Conn.Close()

	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(100, 2)

	m.unregister(c)

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.connections[100]; ok {
		t.Error("user should be removed from connections")
	}
	if _, ok := m.sessions["s1"]; ok {
		t.Error("session should be removed")
	}
	for gid, members := range m.subscriptions {
		if members[100] {
			t.Errorf("user 100 still subscribed to guild %d after unregister", gid)
		}
	}
}

func TestUnregister_IgnoresMismatchedConnection(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	c1 := fakeConn(m, 100, "s1")
	defer c1.Conn.Close()

	// Create a different Connection object for the same user that is NOT registered.
	c2 := &Connection{
		UserID:    100,
		SessionID: "s2",
		Conn:      c1.Conn,
		Send:      make(chan []byte, sendBufferSize),
		manager:   m,
		done:      make(chan struct{}),
	}

	// Unregister c2 (not the actual registered connection).
	m.unregister(c2)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// c1 should still be registered.
	if m.connections[100] != c1 {
		t.Error("original connection should not be removed by mismatched unregister")
	}
}

// ---------------------------------------------------------------------------
// Store Replay Event Tests
// ---------------------------------------------------------------------------

func TestStoreReplayEvent_CreatesBufferOnDemand(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	m.storeReplayEvent(42, Event{Name: "TEST", Data: "data"})

	m.replayMu.RLock()
	defer m.replayMu.RUnlock()

	rb, ok := m.replayBuffer[42]
	if !ok {
		t.Fatal("replay buffer not created for guild 42")
	}

	events := rb.since(0)
	if len(events) != 1 {
		t.Fatalf("replay buffer has %d events, want 1", len(events))
	}
	if events[0].Name != "TEST" {
		t.Errorf("event name = %q, want %q", events[0].Name, "TEST")
	}
}

// ---------------------------------------------------------------------------
// WebSocket Connection Lifecycle Tests
// ---------------------------------------------------------------------------

func setupWSServer(t *testing.T, m *Manager) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/gateway", func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}

		conn := newConnection(ws, m)
		conn.SendPayload(GatewayPayload{
			Op:   OpHello,
			Data: mustMarshal(HelloData{HeartbeatInterval: int(heartbeatInterval.Milliseconds())}),
		})

		go conn.writePump()
		go conn.readPump()
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/gateway"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { ws.Close() })
	return ws
}

func readPayload(t *testing.T, ws *websocket.Conn) GatewayPayload {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var p GatewayPayload
	if err := json.Unmarshal(msg, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return p
}

func sendPayload(t *testing.T, ws *websocket.Conn, p GatewayPayload) {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestWSLifecycle_HelloOnConnect(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	srv := setupWSServer(t, m)
	ws := dialWS(t, srv)

	p := readPayload(t, ws)
	if p.Op != OpHello {
		t.Fatalf("first message op = %d, want %d (HELLO)", p.Op, OpHello)
	}

	var hello HelloData
	if err := json.Unmarshal(p.Data, &hello); err != nil {
		t.Fatalf("unmarshal hello data: %v", err)
	}
	if hello.HeartbeatInterval != int(heartbeatInterval.Milliseconds()) {
		t.Errorf("heartbeat_interval = %d, want %d", hello.HeartbeatInterval, int(heartbeatInterval.Milliseconds()))
	}
}

func TestWSLifecycle_IdentifyAndReady(t *testing.T) {
	tokens := auth.NewTokenService("test-secret")
	token, err := tokens.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	guilds := &mockGuildRepo{
		GetByUserIDFn: func(ctx context.Context, userID int64) ([]models.Guild, error) {
			return []models.Guild{
				{ID: 1, Name: "Guild A"},
				{ID: 2, Name: "Guild B"},
			}, nil
		},
	}

	rdb := newTestRedis(t)
	m := NewManager(tokens, guilds, rdb)
	srv := setupWSServer(t, m)
	ws := dialWS(t, srv)

	// Read HELLO.
	readPayload(t, ws)

	// Send IDENTIFY.
	identifyData := mustMarshal(IdentifyData{Token: token})
	sendPayload(t, ws, GatewayPayload{Op: OpIdentify, Data: identifyData})

	// Read READY.
	p := readPayload(t, ws)
	if p.Op != OpDispatch {
		t.Fatalf("ready op = %d, want %d (DISPATCH)", p.Op, OpDispatch)
	}
	if p.Event == nil || *p.Event != EventReady {
		t.Fatalf("ready event = %v, want %q", p.Event, EventReady)
	}

	var ready ReadyData
	if err := json.Unmarshal(p.Data, &ready); err != nil {
		t.Fatalf("unmarshal ready data: %v", err)
	}
	if ready.UserID != 42 {
		t.Errorf("ready user_id = %d, want 42", ready.UserID)
	}
	if ready.SessionID == "" {
		t.Error("ready session_id should not be empty")
	}
	if len(ready.Guilds) != 2 {
		t.Errorf("ready guilds count = %d, want 2", len(ready.Guilds))
	}

	// Verify the user is subscribed to both guilds.
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, gid := range []int64{1, 2} {
		if !m.subscriptions[gid][42] {
			t.Errorf("user 42 not subscribed to guild %d after IDENTIFY", gid)
		}
	}
}

func TestWSLifecycle_InvalidTokenClosesConnection(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	srv := setupWSServer(t, m)
	ws := dialWS(t, srv)

	readPayload(t, ws)

	identifyData := mustMarshal(IdentifyData{Token: "invalid-token"})
	sendPayload(t, ws, GatewayPayload{Op: OpIdentify, Data: identifyData})

	// The server should close the connection. The next read should fail.
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := ws.ReadMessage()
	if err == nil {
		t.Error("expected read error after invalid identify, got nil")
	}
}

func TestWSLifecycle_HeartbeatExchange(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	srv := setupWSServer(t, m)
	ws := dialWS(t, srv)

	readPayload(t, ws)

	// Send a heartbeat.
	sendPayload(t, ws, GatewayPayload{Op: OpHeartbeat})

	// Should receive heartbeat ACK.
	p := readPayload(t, ws)
	if p.Op != OpHeartbeatAck {
		t.Fatalf("response op = %d, want %d (HEARTBEAT_ACK)", p.Op, OpHeartbeatAck)
	}
}

func TestWSLifecycle_ResumeReplaysEvents(t *testing.T) {
	tokens := auth.NewTokenService("test-secret")

	guilds := &mockGuildRepo{
		GetByUserIDFn: func(ctx context.Context, userID int64) ([]models.Guild, error) {
			return []models.Guild{{ID: 1, Name: "Guild A"}}, nil
		},
	}

	rdb := newTestRedis(t)
	m := NewManager(tokens, guilds, rdb)

	// Pre-populate the replay buffer for guild 1 with 3 events.
	m.storeReplayEvent(1, Event{Name: EventMessageCreate, Data: "msg1"})
	m.storeReplayEvent(1, Event{Name: EventMessageCreate, Data: "msg2"})
	m.storeReplayEvent(1, Event{Name: EventMessageCreate, Data: "msg3"})

	srv := setupWSServer(t, m)
	ws := dialWS(t, srv)

	readPayload(t, ws) // HELLO

	token, err := tokens.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// Send RESUME with sequence 1 → should replay events with seq > 1 (events 2 and 3).
	resumeData := mustMarshal(ResumeData{
		Token:     token,
		SessionID: "old-session",
		Sequence:  1,
	})
	sendPayload(t, ws, GatewayPayload{Op: OpResume, Data: resumeData})

	// Read replayed events.
	var replayed []GatewayPayload
	for i := 0; i < 2; i++ {
		p := readPayload(t, ws)
		replayed = append(replayed, p)
	}

	if len(replayed) != 2 {
		t.Fatalf("replayed %d events, want 2", len(replayed))
	}

	for _, p := range replayed {
		if p.Op != OpDispatch {
			t.Errorf("replayed event op = %d, want %d", p.Op, OpDispatch)
		}
		if p.Event == nil || *p.Event != EventMessageCreate {
			t.Errorf("replayed event name = %v, want %q", p.Event, EventMessageCreate)
		}
	}
}

// ---------------------------------------------------------------------------
// Concurrent Safety Test
// ---------------------------------------------------------------------------

func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	var wg sync.WaitGroup
	for i := int64(0); i < 50; i++ {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			m.SubscribeToGuild(uid, 1)
			m.SubscribeToGuild(uid, 2)
			m.UnsubscribeFromGuild(uid, 1)
		}(i)
	}
	wg.Wait()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// All 50 users should still be in guild 2.
	if len(m.subscriptions[2]) != 50 {
		t.Errorf("guild 2 has %d members, want 50", len(m.subscriptions[2]))
	}
	// Guild 1 should be empty or not exist.
	if members, ok := m.subscriptions[1]; ok && len(members) > 0 {
		t.Errorf("guild 1 still has %d members after all unsubscribes", len(members))
	}
}

func TestConcurrentDispatch(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})

	conns := make([]*Connection, 10)
	for i := range conns {
		uid := int64(i + 1)
		conns[i] = fakeConn(m, uid, "s"+string(rune('0'+i)))
		defer conns[i].Conn.Close()
		m.SubscribeToGuild(uid, 1)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.DispatchToGuild(1, EventMessageCreate, n)
		}(i)
	}
	wg.Wait()

	// Each connection should have received 100 events.
	time.Sleep(50 * time.Millisecond)
	for i, c := range conns {
		events := drainEvents(c)
		if len(events) != 100 {
			t.Errorf("conn %d received %d events, want 100", i, len(events))
		}
	}
}
