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
