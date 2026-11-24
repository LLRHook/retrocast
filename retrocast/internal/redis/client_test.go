package redis

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// pausedRedisProxy lets NewClient complete its real handshake before withholding
// subsequent commands. Unlike a refused socket, an established silent connection
// exercises both go-redis context deadlines and the rate-check's own deadline.
func newSilentRedis(t *testing.T) *Client {
	t.Helper()
	upstream := miniredis.RunT(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var paused atomic.Bool
	release := make(chan struct{})
	var workers sync.WaitGroup
	workers.Go(func() {
		for {
			client, err := listener.Accept()
			if err != nil {
				return
			}
			server, err := net.Dial("tcp", upstream.Addr())
			if err != nil {
				_ = client.Close()
				return
			}
			workers.Go(func() {
				defer func() { _ = client.Close(); _ = server.Close() }()
				done := make(chan struct{})
				go func() { _, _ = io.Copy(client, server); close(done) }()
				buf := make([]byte, 4096)
				for {
					n, err := client.Read(buf)
					if n > 0 {
						if paused.Load() {
							<-release
						}
						if _, e := server.Write(buf[:n]); e != nil {
							break
						}
					}
					if err != nil {
						break
					}
				}
				_ = server.Close()
				<-done
			})
		}
	})
	t.Cleanup(func() { _ = listener.Close(); close(release); workers.Wait() })
	client, err := NewClient("redis://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	paused.Store(true)
	return client
}

func TestCheckRateLimit_DeadlineOnSilentConnection(t *testing.T) {
	client := newSilentRedis(t)
	// The caller's generous deadline bounds the deadline-removal mutant, while
	// the production two-second deadline must win for this check.
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	start := time.Now()
	allowed, _, _, err := client.CheckRateLimit(ctx, "silent", 5, time.Minute)
	elapsed := time.Since(start)
	if err == nil || allowed {
		t.Fatalf("silent Redis: allowed=%v err=%v", allowed, err)
	}
	if elapsed < 1500*time.Millisecond || elapsed > 2750*time.Millisecond {
		t.Fatalf("rate-check deadline: elapsed %s, want approximately 2s (1.5-2.75s)", elapsed)
	}
	t.Logf("silent connection returned in %s", elapsed)
}

// A refused socket can return quickly even without a command deadline. Keep an
// established connection silent so these tests prove the two-second bound.
func TestSessionRedisDeadlinesOnSilentConnection(t *testing.T) {
	for _, operation := range []struct {
		name string
		call func(context.Context, *Client) error
	}{
		{"epoch", func(ctx context.Context, c *Client) error { return c.AdvanceRefreshGeneration(ctx, 25) }},
		{"store", func(ctx context.Context, c *Client) error {
			return c.StoreRefreshToken(ctx, "fixture", 25, time.Minute)
		}},
		{"logout", func(ctx context.Context, c *Client) error { return c.DeleteRefreshToken(ctx, "fixture") }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			client := newSilentRedis(t)
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			start := time.Now()
			err := operation.call(ctx, client)
			elapsed := time.Since(start)
			if err == nil {
				t.Fatal("silent Redis operation must fail")
			}
			if elapsed < 1500*time.Millisecond || elapsed > 2750*time.Millisecond {
				t.Fatalf("session deadline elapsed %s; want approximately 2s", elapsed)
			}
		})
	}
}
