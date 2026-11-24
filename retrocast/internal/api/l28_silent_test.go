package api

import (
	"github.com/alicebob/miniredis/v2"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

func newSilentRedis(t *testing.T) *redisclient.Client {
	t.Helper()
	return newSilentRedisAt(t, miniredis.RunT(t).Addr())
}

func newSilentRedisAt(t *testing.T, upstreamAddr string) *redisclient.Client {
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
			server, err := net.Dial("tcp", upstreamAddr)
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
	client, err := redisclient.NewClient("redis://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	paused.Store(true)
	return client
}
