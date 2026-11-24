package gateway

import (
	"context"
	"encoding/json"
	"github.com/victorivanov/retrocast/internal/models"
	"testing"
)

func TestIdentifyRetriesMembershipAfterJoin(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(map[bool]string{false: "no-other-device", true: "live-other-device"}[live], func(t *testing.T) {
			var m *Manager
			calls := 0
			guilds := []models.Guild{{ID: 1}}
			repo := &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) {
				calls++
				snapshot := append([]models.Guild(nil), guilds...)
				if calls == 1 {
					guilds = append(guilds, models.Guild{ID: 2})
					m.SubscribeToGuild(100, 2)
				}
				return snapshot, nil
			}}
			m = newTestManager(t, repo)
			var other *Connection
			if live {
				other = newConnection(nil, m)
				other.UserID = 100
				other.SessionID = "other"
				m.register(other)
				m.SubscribeToGuild(100, 1)
				defer other.Close()
			}
			c := newConnection(nil, m)
			defer c.Close()
			token, _ := m.tokens.GenerateAccessToken(100)
			raw, _ := json.Marshal(IdentifyData{Token: token})
			m.handleIdentify(c, raw)
			events := drainEvents(c)
			if len(events) == 0 || *events[0].Event != EventReady {
				t.Fatal("missing READY")
			}
			var ready ReadyData
			_ = json.Unmarshal(events[0].Data, &ready)
			if calls != 2 || len(ready.Guilds) != 2 {
				t.Fatalf("join did not retry snapshot: lookups=%d guilds=%v", calls, ready.Guilds)
			}
			if other != nil {
				drainEvents(other)
			}
			m.DispatchToGuild(2, EventGuildUpdate, "joined guild")
			if len(drainEvents(c)) != 1 {
				t.Fatal("new connection missed joined guild")
			}
			if other != nil && len(drainEvents(other)) != 1 {
				t.Fatal("existing device missed joined guild")
			}
		})
	}
}
func TestJoinWhileAwayWithLiveDeviceGetsReady(t *testing.T) {
	guilds := []models.Guild{{ID: 1}}
	m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return guilds, nil }})
	old := newConnection(nil, m)
	old.UserID = 100
	old.SessionID = "old"
	m.register(old)
	defer old.Close()
	live := newConnection(nil, m)
	live.UserID = 100
	live.SessionID = "live"
	m.register(live)
	defer live.Close()
	m.SubscribeToGuild(100, 1)
	m.DispatchToUser(100, EventMessageCreate, "seen")
	drainEvents(old)
	drainEvents(live)
	m.unregister(old)
	guilds = append(guilds, models.Guild{ID: 2})
	m.SubscribeToGuild(100, 2)
	m.DispatchToGuild(2, EventGuildMemberAdd, "joined")
	if len(drainEvents(live)) != 1 {
		t.Fatal("live device was not subscribed in place")
	}
	c := newConnection(nil, m)
	defer c.Close()
	token, _ := m.tokens.GenerateAccessToken(100)
	raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "old", Sequence: 1})
	m.handleResume(c, raw)
	events := drainEvents(c)
	if len(events) == 0 || *events[0].Event != EventReady || c.SessionID == "old" {
		t.Fatal("away device silently resumed after join")
	}
	var ready ReadyData
	_ = json.Unmarshal(events[0].Data, &ready)
	if len(ready.Guilds) != 2 {
		t.Fatal("READY omitted joined guild")
	}
	m.DispatchToGuild(2, EventGuildUpdate, "new guild")
	if len(drainEvents(c)) != 1 {
		t.Fatal("resynced device missed new guild")
	}
}
func TestIdentifyJoinAndRemovalRetryBound(t *testing.T) {
	var m *Manager
	calls := 0
	m = newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) {
		calls++
		if calls%2 == 1 {
			m.SubscribeToGuild(100, 2)
		} else {
			m.UnsubscribeFromGuild(100, 2)
		}
		return []models.Guild{{ID: 1}}, nil
	}})
	c := newConnection(nil, m)
	defer c.Close()
	token, _ := m.tokens.GenerateAccessToken(100)
	raw, _ := json.Marshal(IdentifyData{Token: token})
	m.handleIdentify(c, raw)
	if calls != 3 || len(drainEvents(c)) != 0 {
		t.Fatalf("join/removal retries not bounded: %d", calls)
	}
	select {
	case <-c.done:
	default:
		t.Fatal("exhausted identify stayed open")
	}
}
