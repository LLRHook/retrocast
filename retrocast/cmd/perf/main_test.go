package main

import (
	"testing"
	"time"
)

func TestPeerReceipts(t *testing.T) {
	m := &measurement{pending: map[string]*delivery{"load-test": {sender: 0, sent: time.Now().Add(-time.Millisecond), seen: map[int]bool{}}}}
	m.receive(0, "load-test")
	m.receive(1, "load-test")
	m.receive(1, "load-test")
	m.receive(2, "load-unknown")
	if len(m.latencies) != 1 || m.duplicates != 1 || m.unexpected != 1 {
		t.Fatalf("receipts=%d duplicates=%d unexpected=%d", len(m.latencies), m.duplicates, m.unexpected)
	}
}
func TestP95NearestRank(t *testing.T) {
	values := make([]float64, 100)
	for i := range values {
		values[i] = float64(100 - i)
	}
	if p95(values) != 95 {
		t.Fatal("p95 rank")
	}
}

func TestMissingLatencyIsUnavailable(t *testing.T) {
	if quantile(nil) != nil {
		t.Fatal("missing samples must not be reported as zero milliseconds")
	}
}
