package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"testing"
)

func TestPermissionFailureForcesResyncAndRetries(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }})
	calls := 0
	m.SetPermissionChecker(PermissionCheckerFunc(func(context.Context, int64, int64, int64, permissions.Permission) error {
		calls++
		if calls == 1 {
			return errors.New("transient database failure")
		}
		return nil
	}))
	old := newConnection(nil, m)
	old.UserID = 100
	old.SessionID = "old"
	m.register(old)
	m.SubscribeToGuild(100, 1)
	m.DispatchToUser(100, EventMessageCreate, "seen")
	drainEvents(old)
	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"channel_id": "50"})
	if len(m.visibility) != 0 || m.permissionFailures.Load() != 1 || old.replayFloor <= old.sequence.Load() {
		t.Fatal("transient failure was cached or did not force resync")
	}
	select {
	case <-old.done:
	default:
		t.Fatal("failed session still open")
	}
	m.unregister(old)
	c := newConnection(nil, m)
	defer c.Close()
	token, _ := m.tokens.GenerateAccessToken(100)
	raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "old", Sequence: 1})
	m.handleResume(c, raw)
	events := drainEvents(c)
	if len(events) == 0 || *events[0].Event != EventReady || c.SessionID == "old" {
		t.Fatal("failure did not get fresh READY")
	}
	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"channel_id": "50"})
	if len(drainEvents(c)) != 1 || calls != 2 {
		t.Fatal("healthy next lookup/event not delivered")
	}
}
func TestIdentifyRetriesMembershipAfterRemoval(t *testing.T) {
	for _, always := range []bool{false, true} {
		t.Run(map[bool]string{false: "one-removal", true: "bounded-retries"}[always], func(t *testing.T) {
			var m *Manager
			calls := 0
			repo := &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) {
				calls++
				if calls == 1 || always {
					m.UnsubscribeFromGuild(100, 1)
					return []models.Guild{{ID: 1}}, nil
				}
				return []models.Guild{}, nil
			}}
			m = newTestManager(t, repo)
			c := newConnection(nil, m)
			defer c.Close()
			token, _ := m.tokens.GenerateAccessToken(100)
			raw, _ := json.Marshal(IdentifyData{Token: token})
			m.handleIdentify(c, raw)
			events := drainEvents(c)
			if always {
				if calls != 3 || len(events) != 0 {
					t.Fatal("identify retry not bounded")
				}
				return
			}
			if calls != 2 || len(events) != 1 {
				t.Fatalf("lookup retries=%d events=%d", calls, len(events))
			}
			var ready ReadyData
			_ = json.Unmarshal(events[0].Data, &ready)
			if len(ready.Guilds) != 0 {
				t.Fatal("READY retained kicked guild")
			}
			for _, name := range []string{EventGuildUpdate, EventGuildRoleCreate, EventGuildRoleUpdate} {
				m.DispatchToGuild(1, name, "after removal")
			}
			if len(drainEvents(c)) != 0 {
				t.Fatal("removed member received guild-wide events")
			}
		})
	}
}
func TestRoleUpdateInvalidatesVisibilityAndReplay(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }})
	p := &visibilityChecker{allow: map[int64]bool{100: true}}
	m.SetPermissionChecker(p)
	old := fakeConn(m, 100, "old")
	defer old.Close()
	m.SubscribeToGuild(100, 1)
	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"channel_id": "50"})
	drainEvents(old)
	m.unregister(old)
	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"channel_id": "50", "content": "old secret"})
	p.allow[100] = false
	m.DispatchToGuild(1, EventGuildRoleUpdate, "deny view")
	m.DispatchToGuild(1, EventMessageCreate, map[string]string{"channel_id": "50"})
	if p.calls != 2 {
		t.Fatal("role event did not invalidate cache")
	}
	token, _ := m.tokens.GenerateAccessToken(100)
	raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "old", Sequence: 1})
	c := newConnection(nil, m)
	defer c.Close()
	m.handleResume(c, raw)
	events := drainEvents(c)
	if len(events) == 0 || *events[0].Event != EventReady {
		t.Fatal("authorization revocation replayed old history")
	}
	for _, e := range events {
		if e.Event != nil && *e.Event == EventMessageCreate {
			t.Fatal("secret history replayed")
		}
	}
}
func TestResumeMembershipAndLiveSessionBoundaries(t *testing.T) {
	for _, kind := range []string{"kicked-no-traffic", "kicked-with-traffic", "joined-away", "live"} {
		t.Run(kind, func(t *testing.T) {
			guilds := []models.Guild{{ID: 1}}
			m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return guilds, nil }})
			old := newConnection(nil, m)
			old.UserID = 100
			old.SessionID = "old"
			m.register(old)
			defer old.Close()
			m.SubscribeToGuild(100, 1)
			m.DispatchToUser(100, EventMessageCreate, "seen")
			drainEvents(old)
			if kind != "live" {
				m.unregister(old)
			}
			switch kind {
			case "kicked-no-traffic", "kicked-with-traffic":
				guilds = nil
				m.UnsubscribeFromGuild(100, 1)
				if kind == "kicked-with-traffic" {
					m.DispatchToUser(100, EventMessageCreate, "private replay before fallback")
				}
			case "joined-away":
				guilds = append(guilds, models.Guild{ID: 2})
				m.SubscribeToGuild(100, 2)
			}
			token, _ := m.tokens.GenerateAccessToken(100)
			raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "old", Sequence: 1})
			c := newConnection(nil, m)
			defer c.Close()
			m.handleResume(c, raw)
			events := drainEvents(c)
			if len(events) == 0 || *events[0].Event != EventReady || c.SessionID == "old" {
				t.Fatal("missing fresh READY")
			}
			if kind == "joined-away" {
				m.DispatchToGuild(2, EventGuildUpdate, "new guild")
				if len(drainEvents(c)) != 1 {
					t.Fatal("new guild not subscribed")
				}
			}
			if kind == "live" {
				select {
				case <-old.done:
					t.Fatal("original active session closed")
				default:
				}
			}
		})
	}
}
func TestIdentifyKeepsBothDevices(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }})
	token, _ := m.tokens.GenerateAccessToken(100)
	raw, _ := json.Marshal(IdentifyData{Token: token})
	a := newConnection(nil, m)
	b := newConnection(nil, m)
	defer a.Close()
	defer b.Close()
	m.handleIdentify(a, raw)
	m.handleIdentify(b, raw)
	drainEvents(a)
	drainEvents(b)
	m.DispatchToGuild(1, EventMessageCreate, "both")
	if len(drainEvents(a)) != 1 || len(drainEvents(b)) != 1 {
		t.Fatal("second IDENTIFY dropped first device")
	}
}
