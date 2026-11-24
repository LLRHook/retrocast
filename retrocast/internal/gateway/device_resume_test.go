package gateway

import (
	"context"
	"encoding/json"
	"github.com/victorivanov/retrocast/internal/models"
	"testing"
)

func TestOtherDeviceReconnectPreservesAwayReplay(t *testing.T) {
	for _, method := range []string{"IDENTIFY", "RESUME"} {
		t.Run(method, func(t *testing.T) {
			m := newTestManager(t, &mockGuildRepo{GetByUserIDFn: func(context.Context, int64) ([]models.Guild, error) { return []models.Guild{{ID: 1}}, nil }})
			a := newConnection(nil, m)
			a.UserID = 100
			a.SessionID = "device-a"
			m.register(a)
			defer a.Close()
			b := newConnection(nil, m)
			b.UserID = 100
			b.SessionID = "device-b"
			m.register(b)
			defer b.Close()
			m.SubscribeToGuild(100, 1)
			m.DispatchToUser(100, EventMessageCreate, "seen")
			drainEvents(a)
			drainEvents(b)
			m.unregister(a)
			m.DispatchToGuild(1, EventMessageCreate, "missed-1")
			m.DispatchToGuild(1, EventMessageCreate, "missed-2")
			drainEvents(b)
			m.unregister(b)
			token, _ := m.tokens.GenerateAccessToken(100)
			replacement := newConnection(nil, m)
			defer replacement.Close()
			if method == "IDENTIFY" {
				raw, _ := json.Marshal(IdentifyData{Token: token})
				m.handleIdentify(replacement, raw)
			} else {
				raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "device-b", Sequence: 3})
				m.handleResume(replacement, raw)
				if replacement.SessionID != "device-b" {
					t.Fatal("device B unexpectedly lost its own session")
				}
			}
			resumed := newConnection(nil, m)
			defer resumed.Close()
			raw, _ := json.Marshal(ResumeData{Token: token, SessionID: "device-a", Sequence: 1})
			m.handleResume(resumed, raw)
			events := drainEvents(resumed)
			if resumed.SessionID != "device-a" {
				t.Fatal("another device reconnect invalidated in-window session")
			}
			if len(events) < 2 {
				t.Fatal("missed history was not replayed")
			}
			for i, p := range events {
				if p.Event == nil || *p.Event == EventReady {
					t.Fatal("in-window resume received READY")
				}
				if p.Sequence == nil || *p.Sequence != int64(i+2) {
					t.Fatal("replay sequences changed or reordered")
				}
			}
			for i, want := range []string{"missed-1", "missed-2"} {
				var got string
				_ = json.Unmarshal(events[i].Data, &got)
				if *events[i].Event != EventMessageCreate || got != want {
					t.Fatalf("replay[%d]=%s, want %s", i, got, want)
				}
			}
		})
	}
}
