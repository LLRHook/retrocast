package gateway

import (
	"sync"
	"testing"
)

func TestLiveMembershipRejectsRESTOnlySubscription(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	m.SubscribeToGuild(999, 1)
	if len(m.subscriptions) != 0 {
		t.Fatal("REST-only subscription retained")
	}
}

func TestLiveMembershipJoinAndRemoval(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	c := fakeConn(m, 100, "live")
	defer c.Close()
	m.DispatchToGuild(1, EventGuildUpdate, "before")
	if len(drainEvents(c)) != 0 {
		t.Fatal("received before joining")
	}
	m.SubscribeToGuild(100, 1)
	m.DispatchToGuild(1, EventGuildUpdate, "joined")
	if len(drainEvents(c)) != 1 {
		t.Fatal("next event missing after join")
	}
	m.UnsubscribeFromGuild(100, 1)
	m.DispatchToGuild(1, EventGuildUpdate, "left")
	if len(drainEvents(c)) != 0 {
		t.Fatal("received after removal")
	}
}

func TestLiveMembershipConcurrentRemoval(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	c := fakeConn(m, 100, "live")
	defer c.Close()
	m.SubscribeToGuild(100, 1)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.DispatchToGuild(1, EventGuildUpdate, "concurrent")
			m.UnsubscribeFromGuild(100, 1)
		}()
	}
	wg.Wait()
	drainEvents(c)
	m.DispatchToGuild(1, EventGuildUpdate, "after")
	if len(drainEvents(c)) != 0 {
		t.Fatal("received after removal completed")
	}
}

func TestRemovalNotifiesAllDevicesAndPreservesOtherGuild(t *testing.T) {
	m := newTestManager(t, &mockGuildRepo{})
	a, b, away := fakeConn(m, 100, "a"), fakeConn(m, 100, "b"), fakeConn(m, 100, "away")
	defer a.Close()
	defer b.Close()
	defer away.Close()
	m.SubscribeToGuild(100, 1)
	m.SubscribeToGuild(100, 2)
	m.DispatchToGuild(1, EventGuildUpdate, "old")
	drainEvents(a)
	drainEvents(b)
	drainEvents(away)
	m.unregister(away)
	m.UnsubscribeFromGuild(100, 1)
	m.DispatchToUser(100, EventGuildDelete, map[string]string{"id": "1"})
	for _, c := range []*Connection{a, b} {
		events := drainEvents(c)
		if len(events) != 1 || events[0].Event == nil || *events[0].Event != EventGuildDelete {
			t.Fatal("removal missing on a device")
		}
	}
	if away.replayFloor <= away.sequence.Load() {
		t.Fatal("away device can replay stale guild events")
	}
	m.DispatchToGuild(1, EventGuildUpdate, "forbidden")
	if len(drainEvents(a))+len(drainEvents(b)) != 0 {
		t.Fatal("removed guild still subscribed")
	}
	m.DispatchToGuild(2, EventGuildUpdate, "other")
	if len(drainEvents(a)) != 1 || len(drainEvents(b)) != 1 {
		t.Fatal("other guild subscription lost")
	}
}
