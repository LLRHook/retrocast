package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"sync"
	"testing"
	"time"
)

type visibilityChecker struct {
	allow map[int64]bool
	calls int
}

func (p *visibilityChecker) RequireChannelPermission(_ context.Context, _, _, uid int64, _ permissions.Permission) error {
	p.calls++
	if p.allow[uid] {
		return nil
	}
	return fmt.Errorf("hidden")
}
func TestLiveVisibilityAndInvalidation(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	p := &visibilityChecker{allow: map[int64]bool{100: true}}
	m.SetPermissionChecker(p)
	a := fakeConn(m, 100, "a")
	b := fakeConn(m, 200, "b")
	defer a.Close()
	defer b.Close()
	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	for _, event := range []string{EventMessageCreate, EventMessageUpdate, EventMessageDelete, EventTypingStart, EventMessageReactionAdd} {
		m.DispatchToGuild(1, event, map[string]string{"channel_id": "50"})
		if len(drainEvents(a)) != 1 || len(drainEvents(b)) != 0 {
			t.Fatal("visibility boundary failed", event)
		}
	}
	if p.calls != 2 {
		t.Fatalf("cache missed: %d lookups", p.calls)
	}
	p.allow[100] = false
	p.allow[200] = true
	m.InvalidateGuildPermissions(1)
	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"channel_id": "50"})
	if len(drainEvents(a)) != 0 || len(drainEvents(b)) != 1 {
		t.Fatal("stale visibility cache")
	}
	m.PrepareChannelDeletion(1, 50)
	p.allow[100] = true
	p.allow[200] = false
	m.DispatchToGuild(1, EventChannelDelete, map[string]string{"id": "50"})
	if len(drainEvents(a)) != 0 || len(drainEvents(b)) != 1 {
		t.Fatal("delete used permissions after cascading overrides")
	}
}
func TestLiveMultipleConnectionsAndConcurrentDelivery(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	a := fakeConn(m, 100, "a")
	b := fakeConn(m, 100, "b")
	defer a.Close()
	defer b.Close()
	m.SubscribeToGuild(100, 1)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.DispatchToGuild(1, EventMessageCreate, "guild")
			m.DispatchToUser(100, EventMessageCreate, "dm")
		}()
	}
	wg.Wait()
	for _, c := range []*Connection{a, b} {
		events := drainEvents(c)
		if len(events) != 40 {
			t.Fatalf("got %d", len(events))
		}
		for i, p := range events {
			if *p.Sequence != int64(i+1) {
				t.Fatal("sequence order")
			}
		}
	}
	m.unregister(a)
	if !m.subscriptions[1][100] || !m.connections[100][b] {
		t.Fatal("closing one removed live membership")
	}
	m.DispatchToGuild(1, EventMessageCreate, "still online")
	if len(drainEvents(b)) != 1 {
		t.Fatal("remaining connection lost events")
	}
	m.UnsubscribeFromGuild(100, 1)
	m.DispatchToGuild(1, EventMessageCreate, "left")
	if len(drainEvents(b)) != 0 || len(drainEvents(a)) != 0 {
		t.Fatal("removal leaked to device")
	}
}
func TestLiveSessionReplayAndFallback(t *testing.T) {
	for _, kind := range []string{"valid", "unknown", "foreign", "overflow", "expired", "revoked", "future"} {
		t.Run(kind, func(t *testing.T) {
			m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }})
			old := fakeConn(m, 100, "old")
			defer old.Close()
			m.SubscribeToGuild(100, 1)
			m.DispatchToUser(100, EventMessageCreate, "seen")
			drainEvents(old)
			m.unregister(old)
			for i := 0; i < 3; i++ {
				m.DispatchToGuild(1, EventMessageCreate, i)
			}
			sid := "old"
			seq := int64(1)
			switch kind {
			case "unknown":
				sid = "unknown"
			case "foreign":
				old.UserID = 200
			case "overflow":
				for i := 0; i < 101; i++ {
					m.DispatchToUser(100, EventMessageCreate, i)
				}
			case "expired":
				old.expires = time.Now().Add(-time.Second)
			case "revoked":
				m.UnsubscribeFromGuild(100, 1)
			case "future":
				seq = 999
			}
			token, _ := m.tokens.GenerateAccessToken(100)
			c := newConnection(nil, m)
			defer c.Close()
			raw, _ := json.Marshal(ResumeData{Token: token, SessionID: sid, Sequence: seq})
			m.handleResume(c, raw)
			events := drainEvents(c)
			if kind == "valid" {
				if c.SessionID != "old" || len(events) != 4 {
					t.Fatalf("valid replay: %s %d", c.SessionID, len(events))
				}
				for i, p := range events {
					if *p.Sequence != int64(i+2) {
						t.Fatal("replay renumbered")
					}
				}
			} else {
				if c.SessionID == sid || len(events) == 0 || *events[0].Event != EventReady || *events[0].Sequence != 1 {
					t.Fatalf("missing fresh READY: %+v", events)
				}
			}
		})
	}
}
func TestLivePresenceReadyAndHeartbeatTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rdb.Close() }()
	m := NewManager(auth.NewTokenService("test-secret"), &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }}, nil, rdb)
	a := fakeConn(m, 100, "a")
	defer a.Close()
	m.SubscribeToGuild(100, 1)
	_ = rdb.SetPresence(context.Background(), 100, "dnd")
	outsider := fakeConn(m, 300, "outsider")
	defer outsider.Close()
	m.SubscribeToGuild(300, 2)
	_ = rdb.SetPresence(context.Background(), 300, "online")
	c := newConnection(nil, m)
	defer c.Close()
	token, _ := m.tokens.GenerateAccessToken(200)
	raw, _ := json.Marshal(IdentifyData{Token: token})
	m.handleIdentify(c, raw)
	events := drainEvents(c)
	var ready ReadyData
	_ = json.Unmarshal(events[0].Data, &ready)
	if len(ready.Presences) != 1 || ready.Presences[0].UserID != 100 || ready.Presences[0].Status != "dnd" {
		t.Fatalf("snapshot %+v", ready.Presences)
	}
	mr.FastForward(4 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); a.handleMessage([]byte(`{"op":1}`)) }()
	}
	wg.Wait()
	if mr.TTL("presence:100") != 5*time.Minute {
		t.Fatal("heartbeat did not extend TTL")
	}
	status, _ := rdb.GetPresence(context.Background(), 100)
	if status != "dnd" {
		t.Fatal("heartbeat changed status")
	}
	mr.FastForward(2 * time.Minute)
	status, _ = rdb.GetPresence(context.Background(), 100)
	if status != "dnd" {
		t.Fatal("presence expired after heartbeats")
	}
}

func TestLiveLastConnectionPresenceGrace(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	a := fakeConn(m, 100, "a")
	b := fakeConn(m, 100, "b")
	observer := fakeConn(m, 200, "o")
	defer a.Close()
	defer b.Close()
	defer observer.Close()
	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	_ = m.redis.SetPresence(context.Background(), 100, "online")
	m.unregister(a)
	time.Sleep(10100 * time.Millisecond)
	if len(drainEvents(observer)) != 0 {
		t.Fatal("closing one connection sent offline")
	}
	m.unregister(b)
	select {
	case raw := <-observer.Send:
		var p GatewayPayload
		_ = json.Unmarshal(raw, &p)
		var data PresenceUpdateData
		_ = json.Unmarshal(p.Data, &data)
		if data.Status != "offline" {
			t.Fatal(data)
		}
	case <-time.After(11 * time.Second):
		t.Fatal("last close did not broadcast offline")
	}
}
func TestLiveConcurrentVisibility(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	m.SetPermissionChecker(&visibilityChecker{allow: map[int64]bool{100: true}})
	a := fakeConn(m, 100, "a")
	b := fakeConn(m, 200, "b")
	defer a.Close()
	defer b.Close()
	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(200, 1)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.DispatchToGuild(1, EventTypingStart, map[string]string{"channel_id": "50"}) }()
	}
	wg.Wait()
	if len(drainEvents(a)) != 20 || len(drainEvents(b)) != 0 {
		t.Fatal("concurrent visibility boundary")
	}
}

func TestLiveConcurrentResumeAndDispatch(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }})
	old := fakeConn(m, 100, "old")
	defer old.Close()
	m.SubscribeToGuild(100, 1)
	m.DispatchToUser(100, EventMessageCreate, "seen")
	drainEvents(old)
	m.unregister(old)
	c := newConnection(nil, m)
	defer c.Close()
	token, _ := m.tokens.GenerateAccessToken(100)
	raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "old", Sequence: 1})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); m.handleResume(c, raw) }()
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			m.DispatchToGuild(1, EventMessageCreate, i)
		}
	}()
	wg.Wait()
	events := drainEvents(c)
	if len(events) != 21 {
		t.Fatalf("lost or duplicated concurrent replay: %d", len(events))
	}
	for i, p := range events {
		if p.Sequence == nil || *p.Sequence != int64(i+2) {
			t.Fatal("concurrent replay broke sequence order")
		}
	}
}
