package snowflake

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestNewGenerator_ValidIDs(t *testing.T) {
	g, err := NewGenerator(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g == nil {
		t.Fatal("expected non-nil generator")
	}
}

func TestNewGenerator_InvalidWorkerID(t *testing.T) {
	_, err := NewGenerator(-1, 0)
	if err == nil {
		t.Fatal("expected error for negative workerID")
	}
	_, err = NewGenerator(32, 0)
	if err == nil {
		t.Fatal("expected error for workerID > 31")
	}
}

func TestNewGenerator_InvalidProcessID(t *testing.T) {
	_, err := NewGenerator(0, -1)
	if err == nil {
		t.Fatal("expected error for negative processID")
	}
	_, err = NewGenerator(0, 32)
	if err == nil {
		t.Fatal("expected error for processID > 31")
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	g, err := NewGenerator(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const count = 10000
	seen := make(map[ID]struct{}, count)
	for range count {
		id := g.Generate()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID: %d", id)
		}
		seen[id] = struct{}{}
	}
}

func TestGenerate_Ordering(t *testing.T) {
	g, err := NewGenerator(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prev := g.Generate()
	for range 1000 {
		curr := g.Generate()
		if curr <= prev {
			t.Fatalf("IDs not monotonically increasing: %d >= %d", prev, curr)
		}
		prev = curr
	}
}

func TestGenerate_Positive(t *testing.T) {
	g, err := NewGenerator(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id := g.Generate()
	if id.Int64() <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestGenerate_ConcurrencySafety(t *testing.T) {
	g, err := NewGenerator(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const goroutines = 100
	const perGoroutine = 1000

	var mu sync.Mutex
	seen := make(map[ID]struct{}, goroutines*perGoroutine)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			local := make([]ID, 0, perGoroutine)
			for range perGoroutine {
				local = append(local, g.Generate())
			}
			mu.Lock()
			for _, id := range local {
				if _, exists := seen[id]; exists {
					t.Errorf("duplicate ID under concurrency: %d", id)
				}
				seen[id] = struct{}{}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
}

func TestExtractTimestamp(t *testing.T) {
	g, err := NewGenerator(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	before := time.Now().Add(-time.Millisecond)
	id := g.Generate()
	after := time.Now().Add(time.Millisecond)

	ts := ExtractTimestamp(id.Int64())
	if ts.Before(before) || ts.After(after) {
		t.Fatalf("extracted timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestID_JSONMarshal(t *testing.T) {
	id := ID(1234567890123456789)

	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	expected := `"1234567890123456789"`
	if string(data) != expected {
		t.Fatalf("expected %s, got %s", expected, string(data))
	}
}

func TestID_JSONUnmarshalString(t *testing.T) {
	var id ID
	err := json.Unmarshal([]byte(`"1234567890123456789"`), &id)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if id.Int64() != 1234567890123456789 {
		t.Fatalf("expected 1234567890123456789, got %d", id)
	}
}

func TestID_JSONUnmarshalNumber(t *testing.T) {
	var id ID
	err := json.Unmarshal([]byte(`42`), &id)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if id.Int64() != 42 {
		t.Fatalf("expected 42, got %d", id)
	}
}

func TestID_JSONRoundTrip(t *testing.T) {
	g, err := NewGenerator(7, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	original := g.Generate()
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if original != decoded {
		t.Fatalf("round trip failed: %d != %d", original, decoded)
	}
}

func TestID_JSONInStruct(t *testing.T) {
	type Message struct {
		ID      ID     `json:"id"`
		Content string `json:"content"`
	}

	msg := Message{ID: ID(9876543210), Content: "hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != msg.ID {
		t.Fatalf("expected ID %d, got %d", msg.ID, decoded.ID)
	}
	if decoded.Content != msg.Content {
		t.Fatalf("expected content %q, got %q", msg.Content, decoded.Content)
	}
}

func TestID_String(t *testing.T) {
	id := ID(42)
	if id.String() != "42" {
		t.Fatalf("expected \"42\", got %q", id.String())
	}
}
